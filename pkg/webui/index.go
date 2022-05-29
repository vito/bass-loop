package webui

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"gocloud.dev/blob"
)

type IndexHandler struct {
	DB    *sql.DB
	Blobs *blob.Bucket
}

type IndexTemplateContext struct {
	Runs []RunTemplateContext
}

func (handler *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zapctx.FromContext(ctx)

	logger.Debug("serving index")
	start := time.Now()
	defer func() {
		logger.Debug("served index", zap.Duration("took", time.Since(start)))
	}()

	runListings, err := models.GetRunListings(ctx, handler.DB)
	if err != nil {
		logger.Error("failed to list runs", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var contexts []RunTemplateContext
	for _, r := range runListings {
		run, err := models.RunByID(ctx, handler.DB, r.ID)
		if err != nil {
			logger.Error("failed to get run", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		thunk, err := run.Thunk(ctx, handler.DB)
		if err != nil {
			logger.Error("failed to get run thunk", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user, err := run.User(ctx, handler.DB)
		if err != nil {
			logger.Error("failed to get run thunk", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Debug("generating avatar", zap.String("thunk", thunk.Digest))

		avatar, err := thunkAvatar(thunk.Digest)
		if err != nil {
			logger.Error("failed to render avatar", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Debug("generated avatar", zap.String("thunk", thunk.Digest))

		contexts = append(contexts, RunTemplateContext{
			Run:    run,
			User:   user,
			Thunk:  thunk,
			Avatar: avatar,
		})
	}

	err = tmpl.ExecuteTemplate(w, "index.tmpl", IndexTemplateContext{
		Runs: contexts,
	})
	if err != nil {
		logger.Error("failed to execute template", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
