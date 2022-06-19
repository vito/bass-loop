package controller

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	svg "github.com/ajstarks/svgo"
	"github.com/vito/bass-loop/controller/run"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/logs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/invaders"
	"go.uber.org/zap"
)

type Controller struct {
	Log   *logs.Logger
	DB    *models.Conn
	Blobs *blobs.Bucket
}

// Home struct
type Home struct {
	Runs []*run.Run `json:"runs"`
}

// Index of homes
// GET
func (c *Controller) Index(ctx context.Context) (*Home, error) {
	logger := c.Log

	logger.Debug("serving index")
	start := time.Now()
	defer func() {
		logger.Debug("served index", zap.Duration("took", time.Since(start)))
	}()

	runListings, err := models.GetRunListings(ctx, c.DB)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}

	home := &Home{}

	for _, r := range runListings {
		model, err := models.RunByID(ctx, c.DB, r.ID)
		if err != nil {
			return nil, fmt.Errorf("get run %s: %w", r.ID, err)
		}

		thunk, err := model.Thunk(ctx, c.DB)
		if err != nil {
			return nil, fmt.Errorf("get run thunk: %w", err)
		}

		user, err := model.User(ctx, c.DB)
		if err != nil {
			return nil, fmt.Errorf("get run user: %w", err)
		}

		avatar, err := thunkAvatar(thunk.Digest)
		if err != nil {
			return nil, fmt.Errorf("run avatar: %w", err)
		}

		run := &run.Run{
			Run:       model,
			User:      user,
			Thunk:     thunk,
			Avatar:    avatar,
			Succeeded: model.Succeeded.Int64 == 1,
		}

		if model.EndTime != nil {
			run.Duration = duration(model.EndTime.Time().Sub(model.StartTime.Time()))
		}

		home.Runs = append(home.Runs, run)
	}

	return home, nil
}

func duration(dt time.Duration) string {
	prec := 1
	sec := dt.Seconds()
	if sec < 10 {
		prec = 2
	} else if sec < 100 {
		prec = 1
	}

	return fmt.Sprintf("%.[2]*[1]fs", sec, prec)
}

func thunkAvatar(thunkDigest string) (string, error) {
	h := fnv.New64a()
	if _, err := h.Write([]byte(thunkDigest)); err != nil {
		return "", err
	}

	invader := &invaders.Invader{}
	invader.Set(rand.New(rand.NewSource(int64(h.Sum64()))))

	avatarSvg := new(bytes.Buffer)
	canvas := svg.New(avatarSvg)

	cellSize := 9
	canvas.Startview(
		cellSize*invaders.Width,
		cellSize*invaders.Height,
		0,
		0,
		cellSize*invaders.Width,
		cellSize*invaders.Height,
	)
	canvas.Group()

	for row := range invader {
		y := row * cellSize

		for col := range invader[row] {
			x := col * cellSize
			shade := invader[row][col]

			var color string
			switch shade {
			case invaders.Background:
				color = "transparent"
			case invaders.Shade1:
				color = "var(--base08)"
			case invaders.Shade2:
				color = "var(--base09)"
			case invaders.Shade3:
				color = "var(--base0A)"
			case invaders.Shade4:
				color = "var(--base0B)"
			case invaders.Shade5:
				color = "var(--base0C)"
			case invaders.Shade6:
				color = "var(--base0D)"
			case invaders.Shade7:
				color = "var(--base0E)"
			default:
				return "", fmt.Errorf("invalid shade: %v", shade)
			}

			canvas.Rect(
				x, y,
				cellSize, cellSize,
				fmt.Sprintf("fill: %s", color),
			)
		}
	}

	canvas.Gend()
	canvas.End()

	return avatarSvg.String(), nil
}
