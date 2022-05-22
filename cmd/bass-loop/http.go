package main

import (
	"context"
	"database/sql"
	"net"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/julienschmidt/httprouter"
	"github.com/vito/bass-loop/pkg/github"
	"github.com/vito/bass-loop/pkg/thunk"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/progrock"
	"go.uber.org/zap"
	"gocloud.dev/blob"
	"golang.org/x/sync/errgroup"
)

// MaxBytes is the maximum size of a request payload.
//
// It is arbitrarily set to 25MB, a value based on GitHub's default payload
// limit.
//
// Bass server servers are not designed to handle unbounded or streaming
// payloads, and sometimes need to buffer the entire request body in order to
// check HMAC signatures, so a reasonable default limit is enforced to help
// prevent DoS attacks.
const MaxBytes = 25 * 1024 * 1024

func httpServe(ctx context.Context, db *sql.DB, logs *blob.Bucket) error {
	return withProgress(ctx, "loop", func(ctx context.Context, vertex *progrock.VertexRecorder) error {
		logger := zapctx.FromContext(ctx)

		dispatches := new(errgroup.Group)

		router := httprouter.New()
		router.Handler("GET", "/runs/:run", &thunk.Handler{
			DB: db,
		})

		if config.GitHubApp.ID != 0 {
			keyContent, err := config.GitHubApp.PrivateKey()
			if err != nil {
				return err
			}

			appsTransport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, config.GitHubApp.ID, keyContent)
			if err != nil {
				return err
			}

			router.Handler("POST", "/api/github/hook", &github.WebhookHandler{
				DB:            db,
				Logs:          logs,
				RunCtx:        ctx,
				AppsTransport: appsTransport,
				WebhookSecret: config.GitHubApp.WebhookSecret,
				Dispatches:    dispatches,
			})
		}

		server := &http.Server{
			Addr:    config.HTTPAddr,
			Handler: http.MaxBytesHandler(router, MaxBytes),
			BaseContext: func(net.Listener) context.Context {
				return ctx
			},
		}

		go func() {
			<-ctx.Done()

			logger.Warn("interrupted; stopping gracefully")

			// just passing ctx along to immediately interrupt everything
			server.Shutdown(ctx)
		}()

		logger.Info("listening",
			zap.String("protocol", "http"),
			zap.String("addr", config.HTTPAddr))

		return server.ListenAndServe()
	})
}
