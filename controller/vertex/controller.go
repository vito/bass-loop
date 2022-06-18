package vertex

import (
	"context"
	"fmt"
	"html/template"

	"github.com/vito/bass-loop/pkg/models"
)

type Controller struct {
	// Dependencies...
}

// Vertex struct
type Vertex struct {
	*models.Vertex

	Duration string
	Lines    []Line
}

type Line struct {
	Number  int
	Content template.HTML
}

// Index of runs
// GET /run
func (c *Controller) Show(ctx context.Context) (*Vertex, error) {
	return nil, fmt.Errorf("not implemented")
}
