package webui

import (
	"bytes"
	"database/sql"
	"html/template"
	"net/http"
	"sort"

	chtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/julienschmidt/httprouter"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"gocloud.dev/blob"
)

type ThunkHandler struct {
	DB    *sql.DB
	Blobs *blob.Bucket
}

type ThunkTemplateContext struct {
	Thunk  bass.Thunk
	Avatar template.HTML
	Runs   []RunTemplateContext
	JSON   template.HTML
}

func (handler *ThunkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zapctx.FromContext(ctx)
	params := httprouter.ParamsFromContext(ctx)
	thunkDigest := params.ByName("thunk")

	thunk, err := models.ThunkByDigest(ctx, handler.DB, thunkDigest)
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

	avatar, err := thunkAvatar(bassThunk)
	if err != nil {
		logger.Error("failed to render avatar", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	runs, err := models.RunsByThunkDigest(ctx, handler.DB, thunkDigest)
	if err != nil {
		logger.Error("failed to get thunk vertexes", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.Time().Before(runs[j].StartTime.Time())
	})

	var runContexts []RunTemplateContext
	for _, run := range runs {
		user, err := run.User(ctx, handler.DB)
		if err != nil {
			logger.Error("failed to get run user", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		runContexts = append(runContexts, RunTemplateContext{
			Run:    run,
			User:   user,
			Thunk:  bassThunk,
			Avatar: avatar,
		})
	}

	buf := new(bytes.Buffer)
	enc := bass.NewEncoder(buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(bassThunk); err != nil {
		logger.Error("failed to encode thunk", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iterator, err := lexer.Tokenise(nil, buf.String())
	if err != nil {
		logger.Error("failed to tokenise JSON", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	formatter := chtml.New(
		chtml.PreventSurroundingPre(false),
		chtml.WithClasses(true),
	)

	hlJSON := new(bytes.Buffer)
	err = formatter.Format(hlJSON, styles.Fallback, iterator)
	if err != nil {
		logger.Error("failed to format JSON", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "thunk.tmpl", &ThunkTemplateContext{
		Thunk:  bassThunk,
		Avatar: avatar,
		Runs:   runContexts,
		JSON:   template.HTML(hlJSON.String()),
	})
	if err != nil {
		logger.Error("failed to execute template", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
