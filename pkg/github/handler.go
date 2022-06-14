package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v43/github"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/proto"
	"github.com/vito/bass/pkg/runtimes"
	"github.com/vito/bass/pkg/zapctx"
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
		subCtx := bass.ForkTrace(h.RootCtx)
		subCtx = zapctx.ToContext(subCtx, logger)

		h.Dispatches.Go(func() error {
			err := h.dispatch(
				subCtx,
				repoEvent.Installation.GetID(),
				repoEvent.Sender,
				repoEvent.Repo,
				eventName,
				deliveryID,
				payloadScope,
			)
			if err != nil {
				logger.Warn("dispatch errored", zap.Error(err))
				cli.WriteError(subCtx, err)
			}

			return nil
		})
	} else {
		logger.Warn("ignoring unknown event")
	}

	return nil
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
