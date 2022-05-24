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
	ThunkName string
	RunID     string
	Avatar    template.HTML
	Vertexes  []Vertex
	Duration  string
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

	var vertexes []Vertex
	for _, v := range vs {
		if strings.Contains(v.Name, "[hide]") {
			continue
		}

		logHTML, err := handler.Blobs.ReadAll(ctx, blobs.VertexHTMLLogKey(v))
		if err != nil && gcerrors.Code(err) != gcerrors.NotFound {
			logger.Error("failed to get vertex log", zap.Error(err))
		}

		var dur time.Duration
		if v.EndTime != nil {
			dur = v.EndTime.Time().Sub(v.StartTime.Time())
		}

		vertexes = append(vertexes, Vertex{
			Vertex:   v,
			Duration: duration(dur),
			LogHTML:  template.HTML(logHTML),
		})
	}

	avatar, err := thunkAvatar(bassThunk)
	if err != nil {
		logger.Error("failed to render avatar", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "run.tmpl", &RunTemplateContext{
		ThunkName: bassThunk.Name(),
		RunID:     run.ID,
		Avatar:    avatar,
		Duration:  duration(run.EndTime.Time().Sub(run.StartTime.Time())),
		Vertexes:  vertexes,
	})
	if err != nil {
		logger.Error("failed to execute template", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
