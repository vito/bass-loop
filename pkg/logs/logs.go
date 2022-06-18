package logs

import (
	"github.com/vito/bass/pkg/bass"
	"go.uber.org/zap"
)

type Logger struct {
	*zap.Logger
}

func New() *Logger {
	return &Logger{bass.Logger()}
}
