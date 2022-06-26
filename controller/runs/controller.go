package run

import (
	"context"
	"fmt"

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
	Run      *present.Run      `json:"run"`
	Vertexes []*present.Vertex `json:"vertexes"`
}

// Show run
// GET /runs/:id
func (c *Controller) Show(ctx context.Context, id string) (props *ShowProps, err error) {
	model, err := models.RunByID(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}

	run, err := present.NewRun(ctx, c.Conn, model)
	if err != nil {
		return nil, fmt.Errorf("present run: %w", err)
	}

	vertexModels, err := models.VertexesByRunID(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get vertexes: %w", err)
	}

	vertexes, err := present.Vertexes(ctx, c.Conn, c.Blobs, vertexModels)
	if err != nil {
		return nil, fmt.Errorf("present vertexes: %w", err)
	}

	return &ShowProps{
		Run:      run,
		Vertexes: vertexes,
	}, nil
}
