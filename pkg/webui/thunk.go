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
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"gocloud.dev/blob"
)

type ThunkHandler struct {
	DB    *sql.DB
	Blobs *blob.Bucket
}

type ThunkTemplateContext struct {
	Thunk  *models.Thunk
	Avatar template.HTML
	JSON   template.HTML
	Runs   []RunTemplateContext
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

	avatar, err := thunkAvatar(thunk.Digest)
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
			Thunk:  thunk,
			Avatar: avatar,
		})
	}

	lexer := lexers.Get("json")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iterator, err := lexer.Tokenise(nil, string(thunk.JSON))
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
		Thunk:  thunk,
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
