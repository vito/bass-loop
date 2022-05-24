package runnel

import (
	"github.com/gliderlabs/ssh"
	"github.com/google/go-github/v43/github"
	"go.uber.org/zap"
)

type userIdKey struct{}
type loggerKey struct{}

type GitHubAuthenticator struct {
	Logger *zap.Logger

	*github.Client
}

func (auth GitHubAuthenticator) Auth(ctx ssh.Context, key ssh.PublicKey) bool {
	logger := auth.Logger.With(
		zap.String("user", ctx.User()),
		zap.String("session", ctx.SessionID()),
	)

	user, _, err := auth.Users.Get(ctx, ctx.User())
	if err != nil {
		logger.Error("failed to get user", zap.Error(err))
		return false
	}

	ctx.SetValue(userIdKey{}, user.GetNodeID())
	setLoggerInContext(ctx, logger)

	keys, _, err := auth.Users.ListKeys(ctx, ctx.User(), nil)
	if err != nil {
		logger.Error("failed to list keys", zap.Error(err))
		return false
	}

	for _, k := range keys {
		logger := logger.With(zap.Int64("id", k.GetID()), zap.String("type", key.Type()))

		pkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k.GetKey()))
		if err != nil {
			logger.Error("failed to parse authorized key", zap.Error(err))
			return false
		}

		if ssh.KeysEqual(pkey, key) {
			logger.Info("keys equal")
			return true
		}
	}

	logger.Warn("rejecting auth", zap.String("remote", ctx.RemoteAddr().String()))

	return false
}

func setLoggerInContext(ctx ssh.Context, logger *zap.Logger) {
	ctx.SetValue(loggerKey{}, logger)
}

func loggerFromContext(ctx ssh.Context) *zap.Logger {
	val := ctx.Value(loggerKey{})
	if val == nil {
		return zap.NewNop()
	}

	return val.(*zap.Logger)
}
