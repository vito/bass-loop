package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v43/github"
	"github.com/opencontainers/go-digest"
	"github.com/vito/bass-loop/pkg/bassgh"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/cfg"
	"github.com/vito/bass-loop/pkg/ghapp"
	"github.com/vito/bass-loop/pkg/logs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass-loop/pkg/runs"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/ioctx"
	"github.com/vito/bass/pkg/proto"
	"github.com/vito/bass/pkg/runtimes"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/progrock"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Controller struct {
	Log       *logs.Logger
	DB        *models.Conn
	Blobs     *blobs.Bucket
	Config    *cfg.Config
	Transport *ghapp.Transport

	externalURL *url.URL
	dispatches  *errgroup.Group
}

const DefaultExternalURL = "http://localhost:3000"

func Load(log *logs.Logger, config *cfg.Config, db *models.Conn, blobs *blobs.Bucket, transport *ghapp.Transport) *Controller {
	e := config.ExternalURL
	if e == "" {
		e = DefaultExternalURL
	}

	externalURL, err := url.Parse(e)
	if err != nil {
		// XXX: Controllers can't return error atm
		panic(err)
	}

	return &Controller{
		Log:       log,
		DB:        db,
		Blobs:     blobs,
		Config:    config,
		Transport: transport,

		externalURL: externalURL,
		dispatches:  new(errgroup.Group),
	}
}

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("integration_id") {
	case "github":
		c.handleGithub(w, r)
	}
}

func (c *Controller) handleGithub(w http.ResponseWriter, r *http.Request) {
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

	err = c.Handle(ctx, eventName, deliveryID, payloadBytes)
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

func (c *Controller) Handle(ctx context.Context, eventName, deliveryID string, payload []byte) error {
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
		c.dispatches.Go(func() error {
			err := c.dispatch(
				zapctx.ToContext(context.Background(), logger),
				repoEvent.Installation.GetID(),
				repoEvent.Sender,
				repoEvent.Repo,
				eventName,
				deliveryID,
				payloadScope,
			)
			if err != nil {
				logger.Warn("dispatch errored", zap.Error(err))
			}

			return nil
		})
	} else {
		logger.Warn("ignoring unknown event")
	}

	return nil
}

func (c *Controller) dispatch(ctx context.Context, instID int64, sender *github.User, repo *github.Repository, eventName, deliveryID string, payloadScope *bass.Scope) error {
	// each concurrent Bass must have its own trace
	ctx = bass.WithTrace(context.Background(), &bass.Trace{})

	// load the user's forwarded runtime pool
	ctx, pool, err := c.withUserPool(ctx, sender)
	if err != nil {
		return fmt.Errorf("user %s (%s) pool: %w", sender.GetLogin(), sender.GetNodeID(), err)
	}
	defer pool.Close()

	ghClient := github.NewClient(&http.Client{
		Transport: ghinstallation.NewFromAppsTransport(c.Transport, instID),
	})

	branch, _, err := ghClient.Repositories.GetBranch(ctx, repo.GetOwner().GetLogin(), repo.GetName(), repo.GetDefaultBranch(), true)
	if err != nil {
		return fmt.Errorf("get branch: %w", err)
	}

	hookThunk := bass.Thunk{
		Cmd: bass.ThunkCmd{
			FS: bassgh.NewFS(ctx, ghClient, repo, branch, "project.bass"),
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

	err = runHook(thunkCtx, hookThunk, bass.NewList(
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

func runHook(ctx context.Context, hookThunk bass.Thunk, args bass.List) error {
	logger := zapctx.FromContext(ctx)

	// track thunk runs separately so we can log them later
	ctx, runs := bass.TrackRuns(ctx)

	module, err := bass.NewBass().Load(ctx, hookThunk)
	if err != nil {
		return fmt.Errorf("load project.bass: %w", err)
	}

	var comb bass.Combiner
	if err := module.GetDecode("github-hook", &comb); err != nil {
		return fmt.Errorf("get github-hook: %w", err)
	}

	logger.Info("calling github-hook")

	call := comb.Call(ctx, args, module, bass.Identity)

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

func (c *Controller) withUserPool(ctx context.Context, user *github.User) (context.Context, *runtimes.Pool, error) {
	logger := zapctx.FromContext(ctx)

	rts, err := models.RuntimesByUserID(ctx, c.DB, user.GetNodeID())
	if err != nil {
		return nil, nil, fmt.Errorf("get runtimes: %w", err)
	}

	pool := &runtimes.Pool{}

	for _, rt := range rts {
		svc, err := models.ServiceByUserIDRuntimeNameService(ctx, c.DB, user.GetNodeID(), rt.Name, runtimes.RuntimeServiceName)
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
