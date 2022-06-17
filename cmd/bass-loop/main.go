package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	"github.com/adrg/xdg"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/vito/bass-loop/migrations"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/ioctx"
	"github.com/vito/bass/pkg/zapctx"
)

func main() {
	logger := bass.Logger()
	ctx := zapctx.ToContext(context.Background(), logger)
	ctx = bass.WithTrace(ctx, &bass.Trace{})
	ctx = ioctx.StderrToContext(ctx, os.Stderr)

	err := root(ctx)
	if err != nil {
		cli.WriteError(ctx, err)
		os.Exit(1)
	}
}

func root(ctx context.Context) error {
	if showHelp {
		help(ctx)
		return nil
	}

	if config.Prof.Port != 0 {
		zapctx.FromContext(ctx).Sugar().Debugf("serving pprof on :%d", config.Prof.Port)

		l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Prof.Port))
		if err != nil {
			return err
		}

		go http.Serve(l, nil)
	}

	if config.Prof.FilePath != "" {
		profFile, err := os.Create(config.Prof.FilePath)
		if err != nil {
			return err
		}

		defer profFile.Close()

		pprof.StartCPUProfile(profFile)
		defer pprof.StopCPUProfile()
	}

	return serve(ctx)
}

func openDB() (*sql.DB, error) {
	if config.SQLitePath == "" {
		defaultPath, err := xdg.DataFile("bass-loop/loop.db")
		if err != nil {
			return nil, fmt.Errorf("xdg: %w", err)
		}

		config.SQLitePath = defaultPath
	}

	db, err := sql.Open("sqlite3", config.SQLitePath+"?cache=shared&mode=rwc&_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys")
	if err != nil {
		return nil, fmt.Errorf("open sqlite3: %w", err)
	}

	instance, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("sqlite3 instance: %w", err)
	}

	migrationsSrc, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("migrations fs: %w", err)
	}

	m, err := migrate.NewWithInstance("fs", migrationsSrc, "sqlite3", instance)
	if err != nil {
		return nil, fmt.Errorf("setup migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}
