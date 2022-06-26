package models

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/adrg/xdg"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vito/bass-loop/migrations"
	"github.com/vito/bass-loop/pkg/cfg"
)

type Conn = sql.DB

func Open(config *cfg.Config) (*Conn, error) {
	db, err := open(config)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return db, nil
}

func open(config *cfg.Config) (*sql.DB, error) {
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
