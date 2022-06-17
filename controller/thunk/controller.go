package thunk

import (
	context "context"
)

type Controller struct {
	// Dependencies...
}

// Thunk struct
type Thunk struct {
	// Fields...
}

// Index of thunks
// GET /thunk
func (c *Controller) Index(ctx context.Context) (thunks []*Thunk, err error) {
	return []*Thunk{}, nil
}

// Show thunk
// GET /thunk/:id
func (c *Controller) Show(ctx context.Context, id int) (thunk *Thunk, err error) {
	return &Thunk{}, nil
}
