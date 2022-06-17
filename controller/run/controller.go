package run

import (
	context "context"
)

type Controller struct {
	// Dependencies...
}

// Run struct
type Run struct {
	// Fields...
}

// Index of runs
// GET /run
func (c *Controller) Index(ctx context.Context) (runs []*Run, err error) {
	return []*Run{}, nil
}

// Show run
// GET /run/:id
func (c *Controller) Show(ctx context.Context, id int) (run *Run, err error) {
	return &Run{}, nil
}
