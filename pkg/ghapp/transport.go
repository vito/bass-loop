package ghapp

import (
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/vito/bass-loop/pkg/cfg"
)

type Transport = ghinstallation.AppsTransport

func New(config *cfg.Config) (*Transport, error) {
	if config.GitHubApp.ID == 0 {
		// this optional so I can just run the app with bud run (which doesn't pass
		// env vars - see https://github.com/livebud/bud/issues/151
		return nil, nil
	}

	keyContent, err := config.GitHubApp.PrivateKey()
	if err != nil {
		return nil, err
	}

	appsTransport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, config.GitHubApp.ID, keyContent)
	if err != nil {
		return nil, err
	}

	return appsTransport, nil
}
