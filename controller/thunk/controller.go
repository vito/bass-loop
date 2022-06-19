package thunk

import (
	"bytes"
	context "context"
	"fmt"
	"sort"

	chtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/vito/bass-loop/controller/run"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/logs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass-loop/pkg/present"
)

type Controller struct {
	Log   *logs.Logger
	Conn  *models.Conn
	Blobs *blobs.Bucket
}

// Thunk struct
type Thunk struct {
	*models.Thunk

	Avatar string     `json:"avatar"`
	JSON   string     `json:"json"`
	Runs   []*run.Run `json:"runs"`
}

// Index of thunks
// GET /thunk
func (c *Controller) Index(ctx context.Context) (thunks []*Thunk, err error) {
	return []*Thunk{}, nil
}

// Show thunk
// GET /thunk/:id
func (c *Controller) Show(ctx context.Context, id string) (thunk *Thunk, err error) {
	model, err := models.ThunkByDigest(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get thunk %s: %w", id, err)
	}

	runs, err := models.RunsByThunkDigest(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get runs: %w", err)
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.Time().Before(runs[j].StartTime.Time())
	})

	thunk = &Thunk{
		Thunk: model,
	}

	avatar, err := present.ThunkAvatar(model.Digest)
	if err != nil {
		return nil, fmt.Errorf("render avatar: %w", err)
	}

	thunk.Avatar = avatar

	for _, runModel := range runs {
		user, err := runModel.User(ctx, c.Conn)
		if err != nil {
			return nil, fmt.Errorf("get run user: %w", err)
		}

		run := &run.Run{
			Run:       runModel,
			User:      user,
			Thunk:     model,
			Avatar:    avatar,
			Succeeded: runModel.Succeeded.Int64 == 1,
		}

		if runModel.EndTime != nil {
			run.Duration = present.Duration(runModel.EndTime.Time().Sub(runModel.StartTime.Time()))
		}

		thunk.Runs = append(thunk.Runs, run)
	}

	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iterator, err := lexer.Tokenise(nil, string(thunk.JSON))
	if err != nil {
		return nil, fmt.Errorf("tokenise: %w", err)
	}

	formatter := chtml.New(
		chtml.PreventSurroundingPre(false),
		chtml.WithClasses(true),
	)

	hlJSON := new(bytes.Buffer)
	err = formatter.Format(hlJSON, styles.Fallback, iterator)
	if err != nil {
		return nil, fmt.Errorf("format json: %w", err)
	}

	thunk.JSON = hlJSON.String()

	return thunk, nil
}
