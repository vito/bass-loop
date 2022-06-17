package controller

import (
	context "context"

	"github.com/vito/bass-loop/controller/thunk"
	"github.com/vito/bass-loop/pkg/db"
)

type Controller struct {
	DB *db.DB
	// Blobs *blob.Bucket
}

// Home struct
type Home struct {
	Thunks []*thunk.Thunk
}

// Index of homes
// GET
func (c *Controller) Index(ctx context.Context) (home *Home, err error) {
	return &Home{}, nil
}
