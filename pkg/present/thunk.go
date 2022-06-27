package present

import (
	"context"
	"fmt"

	"github.com/vito/bass-loop/pkg/models"
)

type Thunk struct {
	Digest string `json:"digest"`
	Avatar string `json:"avatar"`
}

func NewThunk(ctx context.Context, db models.DB, model *models.Thunk) (*Thunk, error) {
	thunk := &Thunk{
		Digest: model.Digest,
	}

	avatar, err := ThunkAvatar(model.Digest)
	if err != nil {
		return nil, fmt.Errorf("render avatar: %w", err)
	}

	thunk.Avatar = avatar

	return thunk, nil
}
