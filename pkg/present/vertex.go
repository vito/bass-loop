package present

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/models"
	"gocloud.dev/gcerrors"
)

type Vertex struct {
	Num      int     `json:"num"`
	Name     string  `json:"name"`
	Duration string  `json:"duration"`
	Lines    []*Line `json:"lines"`
	Cached   bool    `json:"cached"`
	Error    string  `json:"error,omitempty"`
}

type Line struct {
	Content string `json:"content"`
}

func Vertexes(ctx context.Context, conn models.DB, bucket *blobs.Bucket, vertexModels []*models.Vertex) ([]*Vertex, error) {
	vertexes := []*Vertex{}

	sort.Slice(vertexModels, func(i, j int) bool {
		return vertexModels[i].EndTime.Time().Before(vertexModels[j].EndTime.Time())
	})

	for i, model := range vertexModels {
		if strings.Contains(model.Name, "[hide]") {
			continue
		}

		logHTML, err := bucket.ReadAll(ctx, blobs.VertexHTMLLogKey(model))
		if err != nil && gcerrors.Code(err) != gcerrors.NotFound {
			return nil, fmt.Errorf("get vertex log: %w", err)
		}

		var lines []*Line
		for _, content := range strings.Split(string(logHTML), "\n") {
			lines = append(lines, &Line{
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
		if model.EndTime != nil {
			dur = model.EndTime.Time().Sub(model.StartTime.Time())
		}

		vertexes = append(vertexes, &Vertex{
			Num:      i + 1,
			Name:     model.Name,
			Duration: Duration(dur),
			Lines:    lines,
			Cached:   model.Cached == 1,
			Error:    model.Error.String,
		})
	}

	return vertexes, nil
}
