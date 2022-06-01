package main

import (
	"context"
	"database/sql"
	"encoding/json"
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

	"github.com/clarafu/envstruct"
	flag "github.com/spf13/pflag"
	"github.com/vito/bass-loop/migrations"
	"github.com/vito/bass-loop/pkg/runnel"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/ioctx"
	"github.com/vito/bass/pkg/zapctx"
)

type Config struct {
	ExternalURL string `env:"EXTERNAL_URL"`

	HTTPAddr string `env:"HTTP_ADDR"`

	TLSCertPath string `env:"TLS_CERT_PATH"`
	TLSKeyPath  string `env:"TLS_KEY_PATH"`

	// for Let's Encrypt autocert
	TLSDomain string `env:"TLS_DOMAIN"`

	SSH runnel.Server `env:"SSH"`

	SQLitePath  string `env:"SQLITE_PATH"`
	BlobsBucket string `env:"BLOBS_BUCKET"`

	GitHubApp GithubAppConfig `env:"GITHUB_APP"`

	Prof struct {
		Port     int    `env:"PORT"`
		FilePath string `env:"FILE_PATH"`
	} `env:"CPU_PROF"`
}

type GithubAppConfig struct {
	ID                int64  `env:"ID"`
	PrivateKeyPath    string `env:"PRIVATE_KEY_PATH"`
	PrivateKeyContent string `env:"PRIVATE_KEY"`
	WebhookSecret     string `env:"WEBHOOK_SECRET"`
}

func (config GithubAppConfig) PrivateKey() ([]byte, error) {
	if config.PrivateKeyPath != "" {
		return os.ReadFile(config.PrivateKeyPath)
	} else if config.PrivateKeyContent != "" {
		return []byte(config.PrivateKeyContent), nil
	} else {
		return nil, fmt.Errorf("missing GitHub app private key")
	}
}

var config Config

var flags = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

var showHelp, showVersion bool

func init() {
	flags.SetOutput(os.Stdout)
	flags.SortFlags = false

	flags.StringVar(&config.ExternalURL, "external-url", "http://localhost:8080", "canonical public URL for the app")

	flags.StringVar(&config.HTTPAddr, "http", "0.0.0.0:8080", "address on which to listen for HTTP traffic")

	flags.StringVar(&config.TLSDomain, "tls-domain", "", "domain for Let's Encrypt cert generation")
	flags.StringVar(&config.TLSCertPath, "tls-cert", "", "TLS certificate path to use for HTTPS")
	flags.StringVar(&config.TLSKeyPath, "tls-key", "", "TLS private key path to use for HTTPS")

	flags.StringVar(&config.SSH.Addr, "ssh", "0.0.0.0:6455", "address on which to listen for SSH traffic")
	flags.StringVar(&config.SSH.HostKeyPath, "ssh-host-key", "", "path to a private key for the SSH server")

	// this is a path, not a DSN; don't want to expose that level of complexity
	// unless we need to. would rather make sure we're tracking a correct default
	flags.StringVar(&config.SQLitePath, "sqlite", "", `SQLite database path (default "$XDG_DATA_HOME/bass-loop/loop.db")`)

	flags.StringVar(&config.BlobsBucket, "blobs", "", `Blobstore URL for storing logs and other data (default "file://$XDG_DATA_HOME/bass-loop/blobs")`)

	flags.Int64Var(&config.GitHubApp.ID, "github-app-id", 0, "GitHub app ID")
	flags.StringVar(&config.GitHubApp.PrivateKeyPath, "github-app-key", "", "path to GitHub app private key")
	flags.StringVar(&config.GitHubApp.WebhookSecret, "github-app-webhook-secret", "", "secret to verify for GitHub app webhook payloads")

	flags.IntVar(&config.Prof.Port, "profile", 0, "port number to bind for Go HTTP profiling")
	flags.StringVar(&config.Prof.FilePath, "cpu-profile", "", "take a CPU profile and save it to this path")

	flags.BoolVarP(&showVersion, "version", "v", false, "print the version number and exit")
	flags.BoolVarP(&showHelp, "help", "h", false, "show bass usage and exit")
}

func main() {
	logger := bass.Logger()
	ctx := zapctx.ToContext(context.Background(), logger)
	ctx = bass.WithTrace(ctx, &bass.Trace{})
	ctx = ioctx.StderrToContext(ctx, os.Stderr)

	err := flags.Parse(os.Args[1:])
	if err != nil {
		cli.WriteError(ctx, bass.FlagError{
			Err:   err,
			Flags: flags,
		})
		os.Exit(2)
		return
	}

	err = envstruct.Envstruct{
		TagName: "env",
		Parser: envstruct.Parser{
			Delimiter: ",",
			Unmarshaler: func(p []byte, dest interface{}) error {
				switch x := dest.(type) {
				case *string:
					*x = string(p)
					return nil
				case *int, *int32, *int64, *uint, *uint32, *uint64:
					return json.Unmarshal(p, dest)
				default:
					return fmt.Errorf("cannot decode env value into %T", dest)
				}
			},
		},
	}.FetchEnv(&config)
	if err != nil {
		cli.WriteError(ctx, err)
		os.Exit(2)
		return
	}

	err = root(ctx)
	if err != nil {
		cli.WriteError(ctx, err)
		os.Exit(1)
	}
}

func root(ctx context.Context) error {
	if showVersion {
		printVersion(ctx)
		return nil
	}

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
