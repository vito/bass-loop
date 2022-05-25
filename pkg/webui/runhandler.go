package webui

import (
	"database/sql"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

type RunHandler struct {
	DB    *sql.DB
	Blobs *blob.Bucket
}

type RunTemplateContext struct {
	Run       *models.Run
	ThunkName string
	Avatar    template.HTML
	Vertexes  []VertexTemplateContext
}

func (rtc RunTemplateContext) Duration() string {
	if rtc.Run.EndTime != nil {
		return duration(rtc.Run.EndTime.Time().Sub(rtc.Run.StartTime.Time()))
	} else {
		return "..."
	}
}

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
		return vs[i].EndTime.Time().Before(vs[j].EndTime.Time())
	})

	var vertexes []VertexTemplateContext
	for vn, v := range vs {
		if strings.Contains(v.Name, "[hide]") {
			continue
		}

		logHTML, err := handler.Blobs.ReadAll(ctx, blobs.VertexHTMLLogKey(v))
		if err != nil && gcerrors.Code(err) != gcerrors.NotFound {
			logger.Error("failed to get vertex log", zap.Error(err))
		}

		var lines []Line
		for i, content := range strings.Split(string(logHTML), "\n") {
			lines = append(lines, Line{
				Number:  i + 1,
				Content: template.HTML(content),
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

		vertexes = append(vertexes, VertexTemplateContext{
			Num:      vn + 1,
			Vertex:   v,
			Duration: duration(dur),
			Lines:    lines,
		})
	}

	avatar, err := thunkAvatar(bassThunk)
	if err != nil {
		logger.Error("failed to render avatar", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "run.tmpl", &RunTemplateContext{
		Run:       run,
		ThunkName: bassThunk.Name(),
		Avatar:    avatar,
		Vertexes:  vertexes,
	})
	if err != nil {
		logger.Error("failed to execute template", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
