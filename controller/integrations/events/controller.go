package events

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/go-github/v43/github"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/cfg"
	"github.com/vito/bass-loop/pkg/ghapp"
	"github.com/vito/bass-loop/pkg/logs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/proto"
	"github.com/vito/bass/pkg/runtimes"
	"github.com/vito/bass/pkg/zapctx"
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
const ProjectFile = "project.bass"

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
		c.handleGitHub(w, r)
	}
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

func callHook(ctx context.Context, hookThunk bass.Thunk, hookBinding bass.Symbol, args bass.List) error {
	logger := zapctx.FromContext(ctx).With(
		zap.Stringer("thunk", hookThunk),
		zap.Stringer("binding", hookBinding),
	)

	// track thunk runs separately so we can log them later
	ctx, runs := bass.TrackRuns(ctx)

	module, err := bass.NewBass().Load(ctx, hookThunk)
	if err != nil {
		return fmt.Errorf("load project.bass: %w", err)
	}

	var comb bass.Combiner
	if err := module.GetDecode(hookBinding, &comb); err != nil {
		return fmt.Errorf("get %s: %w", hookBinding, err)
	}

	logger.Info("calling hook")

	call := comb.Call(ctx, args, module, bass.Identity)

	if _, err := bass.Trampoline(ctx, call); err != nil {
		return fmt.Errorf("hook: %w", err)
	}

	logger.Info("hook called; waiting on runs")

	err = runs.Wait()
	if err != nil {
		logger.Warn("runs failed", zap.Error(err))
	} else {
		logger.Info("runs succeeded")
	}

	return err
}
