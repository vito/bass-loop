package cfg

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/clarafu/envstruct"
)

type Config struct {
	ExternalURL string `env:"EXTERNAL_URL"`

	HTTPAddr string `env:"HTTP_ADDR"`

	TLSCertPath string `env:"TLS_CERT_PATH"`
	TLSKeyPath  string `env:"TLS_KEY_PATH"`

	// for Let's Encrypt autocert
	TLSDomain string `env:"TLS_DOMAIN"`

	SSH RunnelConfig `env:"SSH"`

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

type RunnelConfig struct {
	Addr           string `env:"SSH_ADDR"`
	HostKeyPath    string `env:"SSH_HOST_KEY_PATH"`
	HostKeyContent string `env:"SSH_HOST_KEY"`
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

func New() (*Config, error) {
	env := &Config{}

	err := envstruct.Envstruct{
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
	}.FetchEnv(env)
	if err != nil {
		return nil, err
	}

	return env, nil
}
