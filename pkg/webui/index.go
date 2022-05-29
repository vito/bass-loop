package webui

import (
	"database/sql"
	"net/http"

	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"gocloud.dev/blob"
)

type IndexHandler struct {
	DB    *sql.DB
	Blobs *blob.Bucket
}

type IndexTemplateContext struct {
	Thunks []ThunkTemplateContext
}

func (handler *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zapctx.FromContext(ctx)

	thunks, err := models.GetThunkListings(ctx, handler.DB)
	if err != nil {
		logger.Error("failed to get run", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var thunkContexts []ThunkTemplateContext
	for _, thunk := range thunks {
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

		thunkContexts = append(thunkContexts, ThunkTemplateContext{
			Thunk:  bassThunk,
			Avatar: avatar,
		})
	}

	err = tmpl.ExecuteTemplate(w, "index.tmpl", IndexTemplateContext{
		Thunks: thunkContexts,
	})
	if err != nil {
		logger.Error("failed to execute template", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
