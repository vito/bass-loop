package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v43/github"
	"github.com/opencontainers/go-digest"
	"github.com/vito/bass-loop/pkg/bassgh"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass-loop/pkg/runs"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/ioctx"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/progrock"
	"go.uber.org/zap"
)

func (c *Controller) handleGitHub(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventName := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	if eventName == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "missing event type")
		return
	}

	payloadBytes, err := github.ValidatePayload(r, []byte(c.Config.GitHubApp.WebhookSecret))
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(w, "missing event type")
		return
	}

	err = c.handleGitHubEvent(ctx, eventName, deliveryID, payloadBytes)
	if err != nil {
		cli.WriteError(ctx, err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
}

type GitHubEvent struct {
	// set on check_suite events
	CheckSuite *github.CheckSuite `json:"check_suite,omitempty"`

	// set on check_run events
	CheckRun *github.CheckRun `json:"check_run,omitempty"`

	// set on pull_request events
	PullRequest *github.PullRequest `json:"pull_request,omitempty"`

	// set on all events
	Repo         *github.Repository   `json:"repository,omitempty"`
	Sender       *github.User         `json:"sender,omitempty"`
	Installation *github.Installation `json:"installation,omitempty"`
}

// RefToLoad determines the ref to use for dispatching the event.
//
// For check_suite events, this is the check_suite.head_sha.
//
// For check_run events, this is the check_run.head_sha.
//
// For pull_request events, this is the pull_request.head.sha.
//
// For every other event, this is the repo's default branch's current sha.
func (event *GitHubEvent) RefToLoad(ctx context.Context, ghClient *github.Client) (string, error) {
	repo := event.Repo

	if event.CheckSuite != nil {
		return event.CheckSuite.GetHeadSHA(), nil
	}

	if event.CheckRun != nil {
		return event.CheckRun.GetHeadSHA(), nil
	}

	if event.PullRequest != nil {
		return event.PullRequest.GetHead().GetSHA(), nil
	}

	branch, _, err := ghClient.Repositories.GetBranch(ctx, repo.GetOwner().GetLogin(), repo.GetName(), repo.GetDefaultBranch(), true)
	if err != nil {
		return "", fmt.Errorf("get branch: %w", err)
	}

	return branch.GetCommit().GetSHA(), nil
}

func (c *Controller) handleGitHubEvent(ctx context.Context, eventName, deliveryID string, payload []byte) error {
	logger := zapctx.FromContext(ctx).With(
		zap.String("event", eventName),
		zap.String("delivery", deliveryID),
	)

	logger.Info("handling")

	var payloadScope *bass.Scope
	err := json.Unmarshal(payload, &payloadScope)
	if err != nil {
		return fmt.Errorf("payload->scope: %w", err)
	}

	var event GitHubEvent
	err = json.Unmarshal(payload, &event)
	if err != nil {
		return fmt.Errorf("unmarshal check suite: %w", err)
	}

	if event.Sender == nil || event.Repo == nil || event.Installation == nil {
		// be defensive just because we don't really know what events we'll receive
		logger.Warn("ignoring unknown event")
		return nil
	}

	logger = logger.With(
		zap.String("sender", event.Sender.GetLogin()),
		zap.String("repo", event.Repo.GetFullName()),
	)

	// handle the rest async
	c.dispatches.Go(func() error {
		defer func() {
			// we're forking a goroutine from a goroutine, so prevent panics from
			// taking down the whole loop
			if err := recover(); err != nil {
				logger.Error("dispatch panic!", zap.Any("recovered", err))
			}
		}()

		err := c.dispatch(
			zapctx.ToContext(context.Background(), logger),
			event,
			eventName,
			deliveryID,
			payloadScope,
		)
		if err != nil {
			logger.Warn("dispatch errored", zap.Error(err))
		}

		return nil
	})

	return nil
}

func (c *Controller) dispatch(ctx context.Context, event GitHubEvent, eventName, deliveryID string, payloadScope *bass.Scope) error {
	// each concurrent Bass must have its own trace
	ctx = bass.WithTrace(context.Background(), &bass.Trace{})

	instID := event.Installation.GetID()
	sender := event.Sender
	repo := event.Repo

	// load the user's forwarded runtime pool
	ctx, pool, err := c.withUserPool(ctx, sender)
	if err != nil {
		return fmt.Errorf("user %s (%s) pool: %w", sender.GetLogin(), sender.GetNodeID(), err)
	}
	defer pool.Close()

	ghClient := github.NewClient(&http.Client{
		Transport: ghinstallation.NewFromAppsTransport(c.Transport, instID),
	})

	ref, err := event.RefToLoad(ctx, ghClient)
	if err != nil {
		return fmt.Errorf("get ref to load: %w", err)
	}

	hookThunk := bass.Thunk{
		Cmd: bass.ThunkCmd{
			FS: bassgh.NewFS(ctx, ghClient, repo, ref, ProjectFile),
		},
	}

	run, err := models.CreateThunkRun(ctx, c.DB, sender, hookThunk)
	if err != nil {
		return fmt.Errorf("create hook thunk run: %w", err)
	}

	progress := cli.NewProgress()

	recorder := progrock.NewRecorder(progress)
	thunkCtx := progrock.RecorderToContext(ctx, recorder)

	rec := recorder.Vertex(digest.Digest("delivery:"+deliveryID), fmt.Sprintf("[delivery] %s %s", eventName, deliveryID))
	logger := bass.LoggerTo(rec.Stderr())
	thunkCtx = zapctx.ToContext(thunkCtx, logger)
	thunkCtx = ioctx.StderrToContext(thunkCtx, rec.Stderr())

	err = callHook(thunkCtx, hookThunk, "github-hook", bass.NewList(
		bass.Bindings{
			"event":   bass.String(eventName),
			"payload": payloadScope,
		}.Scope(),
		(&bassgh.Client{
			ExternalURL: c.externalURL,
			DB:          c.DB,
			GH:          ghClient,
			Blobs:       c.Blobs,
			Sender:      sender,
			Repo:        repo,
		}).Scope(),
	))
	if err != nil {
		cli.WriteError(thunkCtx, err)
	}

	rec.Done(err)

	if completeErr := runs.Record(ctx, c.DB, c.Blobs, run, progress, err == nil); completeErr != nil {
		return fmt.Errorf("failed to complete: %w", completeErr)
	}

	return err
}
