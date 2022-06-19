package thunk

import (
	context "context"
	"fmt"
	"sort"

	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/logs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass-loop/pkg/present"
)

type Controller struct {
	Log   *logs.Logger
	Conn  *models.Conn
	Blobs *blobs.Bucket
}

type ShowProps struct {
	Thunk *present.Thunk `json:"thunk"`
	Runs  []*present.Run `json:"runs"`
	JSON  string         `json:"json_html"`
}

// Show thunk
// GET /thunk/:id
func (c *Controller) Show(ctx context.Context, id string) (props *ShowProps, err error) {
	props = &ShowProps{}

	model, err := models.ThunkByDigest(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get thunk %s: %w", id, err)
	}

	props.Thunk, err = present.NewThunk(ctx, c.Conn, model)
	if err != nil {
		return nil, fmt.Errorf("present thunk: %w", err)
	}

	runModels, err := models.RunsByThunkDigest(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get runs: %w", err)
	}

	sort.Slice(runModels, func(i, j int) bool {
		return runModels[i].StartTime.Time().Before(runModels[j].StartTime.Time())
	})

	props.Runs = []*present.Run{}
	for _, runModel := range runModels {
		run, err := present.NewRun(ctx, c.Conn, runModel)
		if err != nil {
			return nil, fmt.Errorf("present run: %w", err)
		}

		props.Runs = append(props.Runs, run)
	}

	props.JSON, err = present.RenderJSON(model.JSON)
	if err != nil {
		return nil, fmt.Errorf("render thunk JSON: %w", err)
	}

	return props, nil
}
