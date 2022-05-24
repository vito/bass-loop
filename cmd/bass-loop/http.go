package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/julienschmidt/httprouter"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"gocloud.dev/blob"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	"github.com/vito/bass-loop/ico"
	"github.com/vito/bass-loop/js"
	"github.com/vito/bass-loop/pkg/github"
	"github.com/vito/bass-loop/pkg/webui"
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

func httpServe(ctx context.Context, db *sql.DB, blobs *blob.Bucket) error {
	logger := zapctx.FromContext(ctx)

	externalURL, err := url.Parse(config.ExternalURL)
	if err != nil {
		return fmt.Errorf("external url: %w", err)
	}

	dispatches := new(errgroup.Group)

	router := httprouter.New()

	router.ServeFiles("/css/*filepath", http.FS(os.DirFS("css")))
	router.ServeFiles("/js/*filepath", http.FS(js.FS))
	router.ServeFiles("/ico/*filepath", http.FS(ico.FS))

	router.Handler("GET", "/runs/:run", &webui.RunHandler{
		DB:    db,
		Blobs: blobs,
	})

	router.Handler("GET", "/thunks/:thunk", &webui.ThunkHandler{
		DB:    db,
		Blobs: blobs,
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
			ExternalURL:   externalURL,
			DB:            db,
			Blobs:         blobs,
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
		server.Shutdown(context.Background())
	}()

	logger.Info("listening",
		zap.String("protocol", "http"),
		zap.String("addr", config.HTTPAddr))

	if config.TLSDomain != "" {
		return server.Serve(autocert.NewListener(config.TLSDomain))
	} else if config.TLSCertPath != "" && config.TLSKeyPath != "" {
		return server.ListenAndServeTLS(config.TLSCertPath, config.TLSKeyPath)
	} else {
		return server.ListenAndServe()
	}
}
