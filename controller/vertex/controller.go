package vertex

import (
	"context"
	"fmt"

	"github.com/vito/bass-loop/pkg/models"
)

type Controller struct {
	// Dependencies...
}

// Vertex struct
type Vertex struct {
	*models.Vertex

	Num      int    `json:"num"`
	Duration string `json:"duration"`
	Lines    []Line `json:"lines"`
}

type Line struct {
	Num     int    `json:"num"`
	Content string `json:"content"`
}

// Index of runs
// GET /run
func (c *Controller) Show(ctx context.Context) (*Vertex, error) {
	return nil, fmt.Errorf("not implemented")
}
