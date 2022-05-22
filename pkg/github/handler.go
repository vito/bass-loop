package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v43/github"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/runtimes"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type WebhookHandler struct {
	DB            *sql.DB
	RunCtx        context.Context
	WebhookSecret string
	AppsTransport *ghinstallation.AppsTransport
	Dispatches    *errgroup.Group
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	if eventType == "" {
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

	err = h.Handle(ctx, eventType, deliveryID, payloadBytes)
	if err != nil {
		cli.WriteError(ctx, err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
}

type RepoEvent struct {
	Repo         *github.Repository   `json:"repository,omitempty"`
	Installation *github.Installation `json:"installation,omitempty"`
}

func (h *WebhookHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	logger := zapctx.FromContext(ctx).With(zap.String("event", eventType), zap.String("delivery", deliveryID))

	logger.Info("handling")

	var event RepoEvent
	err := json.Unmarshal(payload, &event)
	if err != nil {
		return err
	}

	if event.Repo != nil && event.Installation != nil {
		return h.dispatch(
			ctx,
			event.Installation.GetID(),
			event.Repo.GetOwner().GetLogin(),
			event.Repo.GetName(),
			eventType,
			deliveryID,
			payload,
		)
	} else {
		logger.Warn("ignoring unknown event")
	}

	return nil
}

func (h *WebhookHandler) dispatch(ctx context.Context, instID int64, user, repo string, eventType, deliveryID string, payload []byte) error {
	logger := zapctx.FromContext(ctx)

	runCtx := bass.ForkTrace(h.RunCtx)

	var payloadScope *bass.Scope
	err := json.Unmarshal(payload, &payloadScope)
	if err != nil {
		return fmt.Errorf("payload->scope: %w", err)
	}

	scope := bass.NewStandardScope()
	scope.Set("*delivery-id*", bass.String(deliveryID))
	scope.Set("*event*", bass.String(eventType))
	scope.Set("*payload*", payloadScope)

	ghscope := bass.NewEmptyScope()

	instTransport := ghinstallation.NewFromAppsTransport(h.AppsTransport, instID)
	client := github.NewClient(&http.Client{Transport: instTransport})

	ghscope.Set("start-check",
		bass.Func("start-check", "[thunk name sha]", func(ctx context.Context, thunk bass.Thunk, name, sha string) (bass.Combiner, error) {
			thunkRun, err := models.CreateThunkRun(ctx, h.DB, thunk)
			if err != nil {
				return nil, fmt.Errorf("create thunk run: %w", err)
			}

			run, _, err := client.Checks.CreateCheckRun(ctx, user, repo, github.CreateCheckRunOptions{
				Name:      name,
				HeadSHA:   sha,
				Status:    github.String("in_progress"),
				StartedAt: &github.Timestamp{Time: time.Now()},
			})
			if err != nil {
				return nil, fmt.Errorf("create check run: %w", err)
			}

			return thunk.Start(ctx, bass.Func("handler", "[ok?]", func(ctx context.Context, ok bool) error {
				completedAt := time.Now()

				thunkRun.EndTime = sql.NullInt64{
					Int64: completedAt.UnixNano(),
					Valid: true,
				}

				var conclusion string
				if ok {
					thunkRun.Succeeded = sql.NullInt64{Int64: 1, Valid: true}
					conclusion = "success"
				} else if ctx.Err() != nil {
					thunkRun.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
					conclusion = "cancelled"
				} else {
					thunkRun.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
					conclusion = "failure"
				}

				err = thunkRun.Update(ctx, h.DB)
				if err != nil {
					return fmt.Errorf("update thunk run: %w", err)
				}

				_, _, err := client.Checks.UpdateCheckRun(ctx, user, repo, run.GetID(), github.UpdateCheckRunOptions{
					Name:        name,
					Status:      github.String("completed"),
					Conclusion:  github.String(conclusion),
					CompletedAt: &github.Timestamp{Time: completedAt},
				})
				if err != nil {
					return fmt.Errorf("update check run: %w", err)
				}

				return nil
			}))
		}))

	scope.Set("*github*", ghscope)

	// TODO: load config for user
	cfg, err := bass.LoadConfig(defaultConfigTODO)
	if err != nil {
		cli.WriteError(ctx, err)
		return err
	}

	pool, err := runtimes.NewPool(cfg)
	if err != nil {
		return fmt.Errorf("pool: %w", err)
	}
	runCtx = bass.WithRuntimePool(runCtx, pool)

	h.Dispatches.Go(func() error {
		_, err = bass.EvalString(runCtx, scope, `
		(use (.git (linux/alpine/git)))

		(let [{:repository
					 {:clone-url url
						:default-branch branch
						:pushed-at pushed-at}} *payload*
					sha (git:ls-remote url branch pushed-at)
					src (git:checkout url sha)
					project (load (src/project))]
			(project:github-event *payload* *event* *github*))
	`)
		if err != nil {
			logger.Error("delivery failed", zap.String("delivery", deliveryID), zap.Error(err))
			cli.WriteError(runCtx, err)
			return err
		}

		return err
	})

	return nil
}

var defaultConfigTODO = bass.Config{
	Runtimes: []bass.RuntimeConfig{
		{
			Platform: bass.LinuxPlatform,
			Runtime:  runtimes.BuildkitName,
		},
	},
}
