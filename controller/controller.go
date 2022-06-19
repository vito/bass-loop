package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/logs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass-loop/pkg/present"
	"go.uber.org/zap"
)

type Controller struct {
	Log   *logs.Logger
	DB    *models.Conn
	Blobs *blobs.Bucket

	*present.Workaround
}

// Home struct
type Home struct {
	Runs []*present.Run `json:"runs"`
}

// Index of homes
// GET
func (c *Controller) Index(ctx context.Context) (*Home, error) {
	logger := c.Log

	logger.Debug("serving index")
	start := time.Now()
	defer func() {
		logger.Debug("served index", zap.Duration("took", time.Since(start)))
	}()

	runListings, err := models.GetRunListings(ctx, c.DB)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}

	home := &Home{
		Runs: []*present.Run{},
	}

	for _, r := range runListings {
		model, err := models.RunByID(ctx, c.DB, r.ID)
		if err != nil {
			return nil, fmt.Errorf("get run %s: %w", r.ID, err)
		}

		run, err := present.NewRun(ctx, c.DB, model)
		if err != nil {
			return nil, fmt.Errorf("present run: %w", err)
		}

		home.Runs = append(home.Runs, run)
	}

	return home, nil
}
