package thunk

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
)

type Handler struct {
	DB *sql.DB
}

func (handler *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zapctx.FromContext(ctx)
	params := httprouter.ParamsFromContext(ctx)

	vs, err := models.VertexesByRunID(ctx, handler.DB, params.ByName("run"))
	if err != nil {
		logger.Error("failed to get thunk vertexes", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(vs)
}
