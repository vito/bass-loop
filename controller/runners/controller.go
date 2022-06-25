package runners

import "github.com/vito/bass-loop/pkg/runnel"

type Controller struct {
	*runnel.Server
}

func (c *Controller) Index() string {
	return "ok"
}
