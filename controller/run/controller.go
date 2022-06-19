package run

import (
	context "context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vito/bass-loop/controller/vertex"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/logs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass-loop/pkg/present"
	"go.uber.org/zap"
	"gocloud.dev/gcerrors"
)

type Controller struct {
	Log   *logs.Logger
	Conn  *models.Conn
	Blobs *blobs.Bucket
}

// Run struct
type Run struct {
	*models.Run

	User     *models.User     `json:"user"`
	Thunk    *models.Thunk    `json:"thunk"`
	Vertexes []*vertex.Vertex `json:"vertexes"`

	Succeeded bool   `json:"succeeded"`
	Duration  string `json:"duration"`

	// TODO: might be faster to send this as JSON data instead
	Avatar string `json:"avatar"`
}

// Index of runs
// GET /run
func (c *Controller) Index(ctx context.Context) (runs []*models.Run, err error) {
	return []*models.Run{}, nil
}

// Show run
// GET /run/:id
func (c *Controller) Show(ctx context.Context, id string) (run *Run, err error) {
	logger := c.Log

	model, err := models.RunByID(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}

	thunk, err := model.Thunk(ctx, c.Conn)
	if err != nil {
		return nil, fmt.Errorf("get thunk: %w", err)
	}

	vs, err := models.VertexesByRunID(ctx, c.Conn, id)
	if err != nil {
		return nil, fmt.Errorf("get vertexes: %w", err)
	}

	sort.Slice(vs, func(i, j int) bool {
		return vs[i].EndTime.Time().Before(vs[j].EndTime.Time())
	})

	var vertexes []*vertex.Vertex
	for vn, v := range vs {
		if strings.Contains(v.Name, "[hide]") {
			continue
		}

		logHTML, err := c.Blobs.ReadAll(ctx, blobs.VertexHTMLLogKey(v))
		if err != nil && gcerrors.Code(err) != gcerrors.NotFound {
			logger.Error("failed to get vertex log", zap.Error(err))
		}

		var lines []vertex.Line
		for i, content := range strings.Split(string(logHTML), "\n") {
			lines = append(lines, vertex.Line{
				Num:     i + 1,
				Content: content,
			})
		}

		// trim trailing empty lines
		for i := len(lines) - 1; i >= 0; i-- {
			if lines[i].Content == "" {
				lines = lines[:i]
			} else {
				break
			}
		}

		var dur time.Duration
		if v.EndTime != nil {
			dur = v.EndTime.Time().Sub(v.StartTime.Time())
		}

		vertexes = append(vertexes, &vertex.Vertex{
			Num:      vn + 1,
			Vertex:   v,
			Duration: present.Duration(dur),
			Lines:    lines,
		})
	}

	user, err := model.User(ctx, c.Conn)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	avatar, err := present.ThunkAvatar(thunk.Digest)
	if err != nil {
		return nil, fmt.Errorf("render avatar: %w", err)
	}

	run = &Run{
		Run:       model,
		User:      user,
		Thunk:     thunk,
		Vertexes:  vertexes,
		Succeeded: model.Succeeded.Int64 == 1,
		Avatar:    avatar,
	}

	if model.EndTime != nil {
		run.Duration = present.Duration(model.EndTime.Time().Sub(model.StartTime.Time()))
	}

	return run, nil
}
