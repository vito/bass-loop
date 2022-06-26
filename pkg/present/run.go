package present

import (
	"context"
	"fmt"
	"time"

	"github.com/vito/bass-loop/pkg/models"
)

type Run struct {
	ID string `json:"id"`

	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	Duration    string `json:"duration"`
	Succeeded   bool   `json:"succeeded"`

	User  *User  `json:"user"`
	Thunk *Thunk `json:"thunk"`
}

func NewRun(ctx context.Context, db models.DB, model *models.Run) (*Run, error) {
	userModel, err := model.User(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	thunkModel, err := model.Thunk(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get thunk: %w", err)
	}

	thunk, err := NewThunk(ctx, db, thunkModel)
	if err != nil {
		return nil, fmt.Errorf("present thunk: %w", err)
	}

	run := &Run{
		ID: model.ID,

		StartedAt: model.StartTime.Time().Format(time.RFC3339),
		Succeeded: model.Succeeded.Int64 == 1,

		User:  NewUser(userModel),
		Thunk: thunk,
	}

	if model.EndTime != nil {
		run.CompletedAt = model.EndTime.Time().Format(time.RFC3339)
		run.Duration = Duration(model.EndTime.Time().Sub(model.StartTime.Time()))
	}

	return run, nil
}
