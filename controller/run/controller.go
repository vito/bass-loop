package run

import (
	context "context"

	"github.com/vito/bass-loop/controller/vertex"
	"github.com/vito/bass-loop/pkg/models"
)

type Controller struct {
	// Dependencies...
	*models.Conn
}

// Run struct
type Run struct {
	*models.Run

	User     *models.User     `json:"user"`
	Thunk    *models.Thunk    `json:"thunk"`
	Vertexes []*vertex.Vertex `json:"vertexes"`

	Succeeded bool   `json:"succeeded"`
	Duration  string `json:"duration"`

	// TODO: might be faster to send this as JSON data instead
	Avatar string `json:"avatar"`
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
