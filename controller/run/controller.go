package run

import (
	context "context"

	"github.com/vito/bass-loop/controller/vertex"
	"github.com/vito/bass-loop/pkg/models"
)

type Controller struct {
	// Dependencies...
}

// Run struct
type Run struct {
	Run   *models.Run
	User  *models.User
	Thunk *models.Thunk
	// Avatar   template.HTML
	Vertexes []*vertex.Vertex
}

// Index of runs
// GET /run
func (c *Controller) Index(ctx context.Context) (runs []*models.Run, err error) {
	return []*models.Run{}, nil
}

// Show run
// GET /run/:id
func (c *Controller) Show(ctx context.Context, id int) (run *Run, err error) {
	return &Run{}, nil
}
