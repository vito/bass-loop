package logs

import (
	"github.com/vito/bass/pkg/bass"
	"go.uber.org/zap"
)

type Logger = zap.Logger

func New() *Logger {
	return bass.Logger()
}
