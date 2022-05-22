package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/vito/bass/pkg/bass"
)

func CreateThunkRun(ctx context.Context, db *sql.DB, thunk bass.Thunk) (*ThunkRun, error) {
	sha2, err := thunk.SHA256()
	if err != nil {
		return nil, err
	}

	dbThunk, err := ThunkBySha256(ctx, db, sha2)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dbThunk = &Thunk{
				Sha256:    sha2,
				Sensitive: 0,
			}

			err = dbThunk.Save(ctx, db)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	thunkRun := ThunkRun{
		ID:          id.String(),
		ThunkSha256: sha2,
		StartTime:   int(time.Now().Unix()),
	}

	err = thunkRun.Save(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("save thunk run: %w", err)
	}

	return &thunkRun, nil
}
