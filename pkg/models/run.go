package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-github/v43/github"
	"github.com/vito/bass/pkg/bass"
)

func CreateThunkRun(ctx context.Context, db DB, user *github.User, thunk bass.Thunk, meta map[string]any) (*Run, error) {
	sha2, err := thunk.SHA256()
	if err != nil {
		return nil, err
	}

	payload, err := bass.MarshalJSON(thunk)
	if err != nil {
		return nil, err
	}

	dbUser := User{
		ID:    user.GetNodeID(),
		Login: user.GetLogin(),
	}

	err = dbUser.Upsert(ctx, db)
	if err != nil {
		return nil, err
	}

	dbThunk, err := ThunkByDigest(ctx, db, sha2)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dbThunk = &Thunk{
				Digest: sha2,
				JSON:   payload,
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

	startTime := NewTime(time.Now().UTC())
	thunkRun := Run{
		ID:          id.String(),
		UserID:      user.GetNodeID(),
		ThunkDigest: sha2,
		StartTime:   startTime,
	}

	if meta != nil {
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return nil, err
		}

		thunkRun.Meta = sql.NullString{
			String: string(metaJSON),
			Valid:  true,
		}
	}

	err = thunkRun.Save(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("save thunk run: %w", err)
	}

	return &thunkRun, nil
}
