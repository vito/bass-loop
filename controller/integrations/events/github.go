package events

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v43/github"
	"github.com/opencontainers/go-digest"
	defaultinit "github.com/vito/bass-loop/bass/default-init"
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

const initPath = "bass/init.bass"

func (c *Controller) handleGitHub(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = ioctx.StderrToContext(ctx, os.Stderr)

	eventName := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	logger := c.Log.With(
		zap.String("event", eventName),
		zap.String("delivery", deliveryID),
	)
	ctx = zapctx.ToContext(ctx, logger)

	if eventName == "" {
		logger.Warn("missing event type")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "missing event type")
		return
	}

	payloadBytes, err := github.ValidatePayload(r, []byte(c.Config.GitHubApp.WebhookSecret))
	if err != nil {
		logger.Warn("invalid secret")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(w, "invalid secret")
		return
	}

	err = c.handleGitHubEvent(ctx, eventName, deliveryID, payloadBytes)
	if err != nil {
		logger.Error("failed to handle event", zap.Error(err))
		cli.WriteError(ctx, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
}

type GitHubEventPayload struct {
	// set on many events
	Action *string `json:"action,omitempty"`

	// set on push events
	After *string `json:"after,omitempty"`

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

func (event GitHubEventPayload) Meta() models.Meta {
	meta := models.Meta{
		"sender": models.Meta{
			"login":  event.Sender.GetLogin(),
			"action": event.Action,
		},
	}

	if event.Repo != nil {
		meta["repo"] = models.Meta{
			"name":      event.Repo.GetName(),
			"full_name": event.Repo.GetFullName(),
			"url":       event.Repo.GetHTMLURL(),
		}

		sha := event.SHA()
		if sha != "" {
			meta["commit"] = models.Meta{
				"sha": sha,
				"url": event.Repo.GetHTMLURL() + "/commit/" + sha,
			}
		}

		if event.CheckRun != nil {
			branch := event.CheckRun.GetCheckSuite().GetHeadBranch()
			meta["branch"] = models.Meta{
				"name": branch,
				"url":  event.Repo.GetHTMLURL() + "/tree/" + branch,
			}
		}

		if event.CheckSuite != nil {
			branch := event.CheckSuite.GetHeadBranch()
			meta["branch"] = models.Meta{
				"name": branch,
				"url":  event.Repo.GetHTMLURL() + "/tree/" + branch,
			}
		}

		if event.PullRequest != nil {
			meta["pull_request"] = models.Meta{
				"number": event.PullRequest.GetNumber(),
				"url":    event.PullRequest.GetHTMLURL(),
			}

			branch := event.PullRequest.GetHead().GetRef()
			meta["branch"] = models.Meta{
				"name": branch,
				"url":  event.Repo.GetHTMLURL() + "/tree/" + branch,
			}
		}
	}

	return meta
}

func (event *GitHubEventPayload) SHA() string {
	if event.CheckSuite != nil {
		return event.CheckSuite.GetHeadSHA()
	}

	if event.CheckRun != nil {
		return event.CheckRun.GetHeadSHA()
	}

	if event.PullRequest != nil {
		return event.PullRequest.GetHead().GetSHA()
	}

	if event.After != nil {
		return *event.After
	}

	return ""
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
func (event *GitHubEventPayload) RefToLoad(ctx context.Context, ghClient *github.Client) (string, error) {
	repo := event.Repo

	sha := event.SHA()
	if sha != "" {
		return sha, nil
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

	var event GitHubEventPayload
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

func (c *Controller) dispatch(ctx context.Context, payload GitHubEventPayload, eventName, deliveryID string, payloadScope *bass.Scope) error {
	// calling context is ignored; this outlives the hook handler
	_ = ctx

	// each concurrent Bass must have its own trace
	ctx = bass.WithTrace(context.Background(), &bass.Trace{})

	instID := payload.Installation.GetID()
	sender := payload.Sender
	repo := payload.Repo

	// load the user's forwarded runtime pool
	ctx, pool, err := c.withUserPool(ctx, sender)
	if err != nil {
		return fmt.Errorf("user %s (%s) pool: %w", sender.GetLogin(), sender.GetNodeID(), err)
	}
	defer pool.Close()

	ghClient := github.NewClient(&http.Client{
		Transport: ghinstallation.NewFromAppsTransport(c.Transport, instID),
	})

	ref, err := payload.RefToLoad(ctx, ghClient)
	if err != nil {
		return fmt.Errorf("get ref to load: %w", err)
	}

	repoFS := bassgh.NewFS(ctx, ghClient, repo, ref)

	repoRoot, err := c.checkoutRepo(ctx, repoFS, repo.GetCloneURL(), ref)
	if err != nil {
		return fmt.Errorf("checkout repo: %w", err)
	}

	hookPath, err := repoRoot.Extend(bass.FilePath{Path: HookScript})
	if err != nil {
		return fmt.Errorf("extend repo path: %w", err)
	}

	hookThunk := bass.Thunk{
		Args: []bass.Value{hookPath},
		Stdin: []bass.Value{
			bass.Bindings{
				"event":   bass.String(eventName),
				"payload": payloadScope,
			}.Scope(),
		},
	}

	payloadMeta := payload.Meta()

	run, err := models.CreateThunkRun(ctx, c.DB, sender, hookThunk, models.Meta{
		"github": payloadMeta,
		"event": models.Meta{
			"name":     eventName,
			"delivery": deliveryID,
		},
	})
	if err != nil {
		return fmt.Errorf("create hook thunk run: %w", err)
	}

	tape := progrock.NewTape()
	recorder := progrock.NewRecorder(tape)
	runCtx := progrock.RecorderToContext(ctx, recorder)

	rec := recorder.Vertex(digest.Digest("delivery:"+deliveryID), fmt.Sprintf("[delivery] %s %s", eventName, deliveryID))
	logger := bass.LoggerTo(rec.Stderr(), zap.DebugLevel)
	runCtx = zapctx.ToContext(runCtx, logger)
	runCtx = ioctx.StderrToContext(runCtx, rec.Stderr())

	err = callHook(runCtx, hookThunk, &bassgh.Client{
		ExternalURL: c.externalURL,
		DB:          c.DB,
		GH:          ghClient,
		Blobs:       c.Blobs,
		Sender:      sender,
		Repo:        repo,
		Meta:        payloadMeta,
	})
	if err != nil {
		cli.WriteError(runCtx, err)
	}

	rec.Done(err)

	if completeErr := runs.Record(ctx, c.DB, c.Blobs, run, tape, err == nil); completeErr != nil {
		return fmt.Errorf("failed to complete: %w", completeErr)
	}

	return err
}

func (c *Controller) checkoutRepo(ctx context.Context, repoFS fs.FS, cloneURL, ref string) (bass.Path, error) {
	var initThunk bass.Thunk
	if init, err := repoFS.Open(initPath); err == nil {
		// project has bass/init.bass, use it

		// we don't actually need the content here so close it immediately
		//
		// NB: it'll be cached for the actual run
		if err := init.Close(); err != nil {
			return nil, fmt.Errorf("close %s: %w", initPath, err)
		}

		initThunk = bass.Thunk{
			Args: []bass.Value{
				bass.NewFSPath(repoFS, bass.ParseFileOrDirPath(initPath)),
			},
		}
	} else {
		// project has no init.bass; use the bass-loop default

		initThunk = bass.Thunk{
			Args: []bass.Value{
				bass.NewFSPath(defaultinit.FS, bass.ParseFileOrDirPath("init.bass")),
			},
		}
	}

	mod, err := bass.NewBass().Load(ctx, initThunk)
	if err != nil {
		return nil, fmt.Errorf("initialize repo: %w", err)
	}

	var checkout bass.Combiner
	err = mod.GetDecode("checkout", &checkout)
	if err != nil {
		return nil, fmt.Errorf("get checkout binding: %w", err)
	}

	checkoutRes, err := bass.Trampoline(ctx, checkout.Call(
		ctx,
		bass.NewList(
			bass.String(cloneURL),
			bass.String(ref),
		),
		bass.NewEmptyScope(),
		bass.Identity,
	))
	if err != nil {
		return nil, err
	}

	var checkoutPath bass.Path
	if err := checkoutRes.Decode(&checkoutPath); err != nil {
		return nil, err
	}

	return checkoutPath, nil
}
