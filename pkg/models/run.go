package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

func CreateThunkRun(ctx context.Context, db *sql.DB, digest string) (*Run, error) {
	dbThunk, err := ThunkByDigest(ctx, db, digest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dbThunk = &Thunk{
				Digest:    digest,
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

	thunkRun := Run{
		ID:          id.String(),
		ThunkDigest: digest,
		StartTime:   int(time.Now().UnixNano()),
	}

	err = thunkRun.Save(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("save thunk run: %w", err)
	}

	return &thunkRun, nil
}
