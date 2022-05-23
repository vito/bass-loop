package webui

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	svg "github.com/ajstarks/svgo"
	"github.com/julienschmidt/httprouter"
	"github.com/vito/bass-loop/html"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/invaders"
	"go.uber.org/zap"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

type RunHandler struct {
	DB    *sql.DB
	Blobs *blob.Bucket
}

type Vertex struct {
	*models.Vertex

	Duration string
	LogHTML  template.HTML
}

type RunTemplateContext struct {
	ThunkName string
	RunID     string
	Avatar    template.HTML
	Vertexes  []Vertex
}

var tmpl = template.Must(template.ParseFS(html.FS, "*.tmpl"))

func (handler *RunHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zapctx.FromContext(ctx)
	params := httprouter.ParamsFromContext(ctx)
	runID := params.ByName("run")

	run, err := models.RunByID(ctx, handler.DB, runID)
	if err != nil {
		logger.Error("failed to get run", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thunk, err := run.Thunk(ctx, handler.DB)
	if err != nil {
		logger.Error("failed to get run", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var bassThunk bass.Thunk
	err = bass.UnmarshalJSON(thunk.JSON, &bassThunk)
	if err != nil {
		logger.Error("failed to unmarshal thunk", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vs, err := models.VertexesByRunID(ctx, handler.DB, runID)
	if err != nil {
		logger.Error("failed to get thunk vertexes", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Slice(vs, func(i, j int) bool {
		return vs[i].EndTime.Int64 < vs[j].EndTime.Int64
	})

	var vertexes []Vertex
	for _, v := range vs {
		if strings.Contains(v.Name, "[hide]") {
			continue
		}

		logHTML, err := handler.Blobs.ReadAll(ctx, blobs.VertexHTMLLogKey(v))
		if err != nil {
			if gcerrors.Code(err) == gcerrors.NotFound {
				logger.Debug("no logs", zap.String("digest", v.Digest))
			} else {
				logger.Error("failed to get vertex log", zap.Error(err))
			}
		}

		var dur time.Duration
		if v.EndTime.Valid && v.StartTime.Valid {
			dur = time.Duration(v.EndTime.Int64 - v.StartTime.Int64)
		}

		vertexes = append(vertexes, Vertex{
			Vertex:   v,
			Duration: duration(dur),
			LogHTML:  template.HTML(logHTML),
		})
	}

	avatar, err := handler.renderThunk(bassThunk)
	if err != nil {
		logger.Error("failed to render avatar", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "run.tmpl", &RunTemplateContext{
		ThunkName: bassThunk.Name(),
		RunID:     run.ID,
		Avatar:    avatar,
		Vertexes:  vertexes,
	})
	if err != nil {
		logger.Error("failed to execute template", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func duration(dt time.Duration) string {
	prec := 1
	sec := dt.Seconds()
	if sec < 10 {
		prec = 2
	} else if sec < 100 {
		prec = 1
	}

	return fmt.Sprintf("[%.[2]*[1]fs]", sec, prec)
}

func (handler *RunHandler) renderThunk(thunk bass.Thunk) (template.HTML, error) {
	invader, err := thunk.Avatar()
	if err != nil {
		return "", err
	}

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

	return template.HTML(avatarSvg.String()), nil
}
