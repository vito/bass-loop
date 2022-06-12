package github

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aoldershaw/ansi"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v43/github"
	"github.com/mattn/go-colorable"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/proto"
	"github.com/vito/bass/pkg/runtimes"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/progrock"
	"go.uber.org/zap"
	"gocloud.dev/blob"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WebhookHandler struct {
	ExternalURL   *url.URL
	DB            *sql.DB
	Blobs         *blob.Bucket
	RootCtx       context.Context
	WebhookSecret string
	AppsTransport *ghinstallation.AppsTransport
	Dispatches    *errgroup.Group
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventName := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	if eventName == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "missing event type")
		return
	}

	payloadBytes, err := github.ValidatePayload(r, []byte(h.WebhookSecret))
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(w, "missing event type")
		return
	}

	err = h.Handle(ctx, eventName, deliveryID, payloadBytes)
	if err != nil {
		cli.WriteError(ctx, err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
}

type RepoEvent struct {
	Repo         *github.Repository   `json:"repository"`
	Sender       *github.User         `json:"sender"`
	Installation *github.Installation `json:"installation"`
}

func (h *WebhookHandler) Handle(ctx context.Context, eventName, deliveryID string, payload []byte) error {
	logger := zapctx.FromContext(ctx).With(zap.String("event", eventName), zap.String("delivery", deliveryID))

	logger.Info("handling")

	var payloadScope *bass.Scope
	err := json.Unmarshal(payload, &payloadScope)
	if err != nil {
		return fmt.Errorf("payload->scope: %w", err)
	}

	var repoEvent RepoEvent
	err = json.Unmarshal(payload, &repoEvent)
	if err != nil {
		return err
	}

	logger = logger.With(
		zap.String("sender", repoEvent.Sender.GetLogin()),
	)

	if repoEvent.Repo != nil && repoEvent.Installation != nil {
		h.Dispatches.Go(func() error {
			err := h.dispatch(
				zapctx.ToContext(bass.ForkTrace(h.RootCtx), logger),
				repoEvent.Installation.GetID(),
				repoEvent.Sender,
				repoEvent.Repo,
				eventName,
				deliveryID,
				payloadScope,
			)
			if err != nil {
				logger.Warn("dispatch errored", zap.Error(err))
				cli.WriteError(ctx, err)
			}

			return nil
		})
	} else {
		logger.Warn("ignoring unknown event")
	}

	return nil
}

func (h *WebhookHandler) withUserPool(ctx context.Context, user *github.User) (context.Context, *runtimes.Pool, error) {
	logger := zapctx.FromContext(ctx)

	rts, err := models.RuntimesByUserID(ctx, h.DB, user.GetNodeID())
	if err != nil {
		return nil, nil, fmt.Errorf("get runtimes: %w", err)
	}

	pool := &runtimes.Pool{}

	for _, rt := range rts {
		svc, err := models.ServiceByUserIDRuntimeNameService(ctx, h.DB, user.GetNodeID(), rt.Name, runtimes.RuntimeServiceName)
		if err != nil {
			logger.Error("failed to get service", zap.Error(err))
			pool.Close()
			return nil, nil, fmt.Errorf("get runtime service: %w", err)
		}

		conn, err := grpc.Dial(svc.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error("grpc dial failed", zap.Error(err))
			pool.Close()
			return nil, nil, err
		}

		assoc := runtimes.Assoc{
			Platform: bass.Platform{
				OS:   rt.Os,
				Arch: rt.Arch,
			},
			Runtime: &runtimes.Client{
				Conn:          conn,
				RuntimeClient: proto.NewRuntimeClient(conn),
			},
		}

		pool.Runtimes = append(pool.Runtimes, assoc)
	}

	return bass.WithRuntimePool(ctx, pool), pool, nil

}

func (h *WebhookHandler) dispatch(ctx context.Context, instID int64, sender *github.User, repo *github.Repository, eventName, deliveryID string, payloadScope *bass.Scope) error {
	logger := zapctx.FromContext(ctx)

	// track thunk runs separately so we can log them later
	ctx, runs := bass.TrackRuns(ctx)

	// load the user's forwarded runtime pool
	ctx, pool, err := h.withUserPool(ctx, sender)
	if err != nil {
		return fmt.Errorf("user %s (%s) pool: %w", sender.GetLogin(), sender.GetNodeID(), err)
	}
	defer pool.Close()

	ghClient := github.NewClient(&http.Client{
		Transport: ghinstallation.NewFromAppsTransport(h.AppsTransport, instID),
	})

	branch, _, err := ghClient.Repositories.GetBranch(ctx, repo.GetOwner().GetLogin(), repo.GetName(), repo.GetDefaultBranch(), true)
	if err != nil {
		return fmt.Errorf("get branch: %w", err)
	}

	module, err := bass.NewBass().Load(ctx, bass.Thunk{
		Cmd: bass.ThunkCmd{
			FS: NewGHPath(ctx, ghClient, repo, branch, "project.bass"),
		},
	})
	if err != nil {
		return fmt.Errorf("load project.bass: %w", err)
	}

	var comb bass.Combiner
	if err := module.GetDecode("github-hook", &comb); err != nil {
		return fmt.Errorf("get github-hook: %w", err)
	}

	logger.Info("calling github-hook")

	call := comb.Call(
		ctx,
		bass.NewList(
			bass.Bindings{
				"event":   bass.String(eventName),
				"payload": payloadScope,
			}.Scope(),
			(&BassGitHubClient{
				ExternalURL: h.ExternalURL,
				DB:          h.DB,
				GH:          ghClient,
				Blobs:       h.Blobs,
				Sender:      sender,
				Repo:        repo,
			}).Scope(),
		),
		module,
		bass.Identity,
	)

	if _, err := bass.Trampoline(ctx, call); err != nil {
		return fmt.Errorf("hook: %w", err)
	}

	logger.Info("github-hook called; waiting on runs")

	err = runs.Wait()
	if err != nil {
		logger.Warn("runs failed", zap.Error(err))
	} else {
		logger.Info("runs succeeded")
	}

	return err
}

type BassGitHubClient struct {
	ExternalURL *url.URL
	DB          *sql.DB
	Blobs       *blob.Bucket
	GH          *github.Client
	Sender      *github.User
	Repo        *github.Repository
}

func (client *BassGitHubClient) Scope() *bass.Scope {
	ghscope := bass.NewEmptyScope()
	ghscope.Set("start-check",
		bass.Func("start-check", "[thunk name sha]", client.StartCheck))

	return ghscope
}

func (client *BassGitHubClient) StartCheck(ctx context.Context, thunk bass.Thunk, checkName, sha string) (bass.Combiner, error) {
	logger := zapctx.FromContext(ctx)

	run, err := models.CreateThunkRun(ctx, client.DB, client.Sender, thunk)
	if err != nil {
		return nil, fmt.Errorf("create thunk run: %w", err)
	}

	thunkURL, err := client.ExternalURL.Parse("/thunks/" + thunk.Name())
	if err != nil {
		return nil, fmt.Errorf("create thunk run: %w", err)
	}

	runURL, err := client.ExternalURL.Parse("/runs/" + run.ID)
	if err != nil {
		return nil, fmt.Errorf("create thunk run: %w", err)
	}

	output := &github.CheckRunOutput{
		Title: github.String("(" + thunk.Cmd.ToValue().String() + ")"),
		Summary: github.String(strings.Join([]string{
			`* **thunk** [` + thunk.Name() + `](` + thunkURL.String() + `)`,
			`* **run** [` + run.ID + `](` + runURL.String() + `)`,
			``,
			"```sh",
			"# final command",
			thunk.Cmdline(),
			"```",
		}, "\n")),
	}

	checkRun, _, err := client.GH.Checks.CreateCheckRun(ctx, client.Repo.GetOwner().GetLogin(), client.Repo.GetName(), github.CreateCheckRunOptions{
		Name:       checkName,
		HeadSHA:    sha,
		Status:     github.String("in_progress"),
		StartedAt:  &github.Timestamp{Time: time.Now()},
		ExternalID: github.String(run.ID),
		DetailsURL: github.String(runURL.String()),
		Output:     output,
	})
	if err != nil {
		return nil, fmt.Errorf("create check run: %w", err)
	}

	progress := cli.NewProgress()
	thunkCtx := progrock.RecorderToContext(ctx, progrock.NewRecorder(progress))

	complete := func(ctx context.Context, ok bool) error {
		completedAt := models.NewTime(time.Now().UTC())
		run.EndTime = &completedAt

		var conclusion string
		if ok {
			run.Succeeded = sql.NullInt64{Int64: 1, Valid: true}
			conclusion = "success"
		} else if ctx.Err() != nil {
			run.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
			conclusion = "cancelled"
		} else {
			run.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
			conclusion = "failure"
		}

		err = progress.EachVertex(func(v *cli.Vertex) error {
			var startTime, endTime models.Time
			if v.Started != nil {
				startTime = models.NewTime(v.Started.UTC())
			}
			if v.Completed != nil {
				endTime = models.NewTime(v.Completed.UTC())
			}

			var vErr sql.NullString
			if v.Error != "" {
				vErr.String = v.Error
				vErr.Valid = true
			}

			var cached int
			if v.Cached {
				cached = 1
			}

			vtx := &models.Vertex{
				Digest:    v.Digest.String(),
				RunID:     run.ID,
				Name:      v.Name,
				StartTime: &startTime,
				EndTime:   &endTime,
				Error:     vErr,
				Cached:    cached,
			}

			htmlBuf := new(bytes.Buffer)
			if v.Log.Len() > 0 {
				if err := client.Blobs.WriteAll(ctx, blobs.VertexRawLogKey(vtx), v.Log.Bytes(), nil); err != nil {
					return fmt.Errorf("store raw logs: %w", err)
				}

				var lines ansi.Lines
				writer := ansi.NewWriter(&lines,
					// arbitrary, matched my screen
					ansi.WithInitialScreenSize(67, 316))
				if _, err := writer.Write(v.Log.Bytes()); err != nil {
					return fmt.Errorf("write log: %w", err)
				}

				if err := ANSIHTML.Execute(htmlBuf, lines); err != nil {
					return fmt.Errorf("render html: %w", err)
				}

				if err := client.Blobs.WriteAll(ctx, blobs.VertexHTMLLogKey(vtx), htmlBuf.Bytes(), nil); err != nil {
					return fmt.Errorf("store html logs: %w", err)
				}
			}

			for {
				if err := vtx.Save(ctx, client.DB); err != nil {
					// TODO why is this happening so often even with retrying?
					logger.Error("failed to save vertex", zap.Error(err))
					time.Sleep(time.Second)
					continue
				}

				break
			}

			for _, input := range v.Inputs {
				edge := models.VertexEdge{
					SourceDigest: input.String(),
					TargetDigest: v.Digest.String(),
				}

				_, err := models.VertexEdgeBySourceDigestTargetDigest(ctx, client.DB, edge.SourceDigest, edge.TargetDigest)
				if err != nil && errors.Is(err, sql.ErrNoRows) {
					// this could conflict with another edge, but that's ok; we just do
					// the above check to make the logs less noisy
					if err := edge.Insert(ctx, client.DB); err != nil {
						logger.Warn("insert edge", zap.Error(err))
					}
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("store vertex logs: %w", err)
		}

		err = run.Update(ctx, client.DB)
		if err != nil {
			return fmt.Errorf("update thunk run: %w", err)
		}

		outBuf := new(bytes.Buffer)
		progress.Summarize(colorable.NewNonColorable(outBuf))
		output.Text = github.String("```\n" + outBuf.String() + "\n```")

		_, _, err := client.GH.Checks.UpdateCheckRun(
			ctx,
			client.Repo.GetOwner().GetLogin(),
			client.Repo.GetName(),
			checkRun.GetID(),
			github.UpdateCheckRunOptions{
				Name:        checkName,
				Status:      github.String("completed"),
				Conclusion:  github.String(conclusion),
				CompletedAt: &github.Timestamp{Time: completedAt.Time()},
				Output:      output,
			},
		)
		if err != nil {
			return fmt.Errorf("update check run: %w", err)
		}

		return nil
	}

	return thunk.Start(thunkCtx, bass.Func("handler", "[ok?]", func(ctx context.Context, ok bool) error {
		if err := complete(ctx, ok); err != nil {
			return fmt.Errorf("failed to complete: %w", err)
		}

		if ok {
			return nil
		}

		// bubble up an error so it gets logged
		//
		// might make sense to remove this someday, but I would rather start with
		// too much logging
		return fmt.Errorf("%s: check %s: %s failed", sha, checkName, thunk)
	}))
}

// TODO: support modifiers (bold/etc) - it's a bit tricky, may need changes
// upstream
var ANSIHTML = template.Must(template.New("ansi").Parse(`{{- range . -}}
	<span class="ansi-line">
		{{- range . -}}
		{{- if or .Style.Foreground .Style.Background .Style.Modifier -}}
			<span class="{{with .Style.Foreground}}fg-{{.}}{{end}}{{with .Style.Background}} bg-{{.}}{{end}}">
				{{- printf "%s" .Data -}}
			</span>
		{{- else -}}
			{{- printf "%s" .Data -}}
		{{- end -}}
		{{- end -}}
	</span>
{{end}}`))
