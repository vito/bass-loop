package runnel

import (
	"github.com/vito/bass/pkg/bass"
)

type Pool struct {
	UserID string
}

func (pool Pool) Select(*bass.Platform) (bass.Runtime, error) {
	return nil, nil
}

func (Pool) All() ([]bass.Runtime, error) {
	return nil, nil
}
