package runnel

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/google/go-github/v43/github"
	flag "github.com/spf13/pflag"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	"gocloud.dev/blob"
	gossh "golang.org/x/crypto/ssh"
)

type Server struct {
	Addr           string `env:"SSH_ADDR"`
	HostKeyPath    string `env:"SSH_HOST_KEY_PATH"`
	HostKeyContent string `env:"SSH_HOST_KEY"`

	DB    *sql.DB
	Blobs *blob.Bucket
}

const (
	ForwardCommandName = "forward"
	HelpCommandName    = "help"
)

type Command struct {
	Command  string
	Callback func(ssh.Session, *flag.FlagSet, []string)
}

func (server *Server) ListenAndServe(ctx context.Context) error {
	logger := zapctx.FromContext(ctx)

	commands := []Command{
		{
			Command:  "forward",
			Callback: server.HandleForwardCommand,
		},
	}

	ssh.Handle(func(s ssh.Session) {
		logger.Info("handling ssh session",
			zap.String("user", s.User()),
			zap.Strings("command", s.Command()))

		cmdline := s.Command()

		var cmd string
		if len(cmdline) > 0 {
			cmd = cmdline[0]
		}

		knownCommands := []string{}
		for _, c := range commands {
			knownCommands = append(knownCommands, c.Command)

			if c.Command == cmd {
				c.Callback(s, flag.NewFlagSet(cmd, flag.ContinueOnError), cmdline[1:])
				return
			}
		}

		fmt.Fprintf(s, "unknown command: %q\n", cmd)
		fmt.Fprintf(s, "known commands: %q\n", knownCommands)
		s.Exit(2)
	})

	opts := []ssh.Option{
		ssh.PublicKeyAuth(GitHubAuthenticator{
			Logger: logger,
			Client: github.NewClient(nil),
		}.Auth),
	}

	if server.HostKeyPath != "" {
		opts = append(opts, ssh.HostKeyFile(server.HostKeyPath))
	} else if server.HostKeyContent != "" {
		opts = append(opts, ssh.HostKeyPEM([]byte(server.HostKeyContent)))
	}

	forwardHandler := NewForwardHandler(ctx, server.DB)

	sshServer := &ssh.Server{
		Addr: server.Addr,

		RequestHandlers: map[string]ssh.RequestHandler{
			// "tcpip-forward":        forwardHandler.HandleSSHRequest,
			// "cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			StreamlocalForwardChannelType:       forwardHandler.HandleStreamlocalForward,
			CancelStreamlocalForwardChannelType: forwardHandler.HandleCancelStreamlocalForward,

			KeepaliveRequestType: func(ctx ssh.Context, _ *ssh.Server, _ *gossh.Request) (bool, []byte) {
				logger.Debug("keepalive", zap.String("remote", ctx.RemoteAddr().String()))
				return true, nil
			},

			"default": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
				logger.Warn("unhandled request", zap.String("request", req.Type), zap.String("remote", ctx.RemoteAddr().String()))
				return true, nil
			},
		},
	}

	for _, opt := range opts {
		sshServer.SetOption(opt)
	}

	if _, err := server.DB.Exec(`DELETE FROM runtimes`); err != nil {
		return fmt.Errorf("clean up runtimes: %w", err)
	}

	if _, err := server.DB.Exec(`DELETE FROM services`); err != nil {
		return fmt.Errorf("clean up services: %w", err)
	}

	logger.Info("listening",
		zap.String("protocol", "ssh"),
		zap.String("addr", server.Addr))

	go func() {
		<-ctx.Done()
		logger.Warn("interrupted; stopping gracefully")
		sshServer.Shutdown(context.Background())
	}()

	return sshServer.ListenAndServe()
}

func (server *Server) HandleForwardCommand(s ssh.Session, flags *flag.FlagSet, args []string) {
	logger := bass.LoggerTo(s).With(zap.String("side", "server"))

	var priority int
	flags.IntVarP(&priority, "priority", "p", 0, "priority")

	var os, arch string
	flags.StringVar(&os, "os", "linux", "runtime platform OS (ie. GOOS)")
	flags.StringVar(&arch, "arch", "amd64", "runtime platform architecture (i.e. GOARCH)")

	if err := flags.Parse(args); err != nil {
		logger.Error("failed to parse flags", zap.Error(err))
		s.Exit(2)
		return
	}

	userIDVal := s.Context().Value(userIdKey{})
	if userIDVal == nil {
		logger.Error("user id not found in context")
		s.Exit(1)
		return
	}

	userID := userIDVal.(string)

	runtime := models.Runtime{
		UserID:    userID,
		Name:      s.Context().SessionID(),
		Priority:  priority,
		Os:        os,
		Arch:      arch,
		ExpiresAt: models.NewTime(time.Now().Add(time.Hour).UTC()),
	}

	if err := runtime.Insert(s.Context(), server.DB); err != nil {
		logger.Error("failed to save runtime", zap.Error(err))
		s.Exit(1)
		return
	}

	logger.Info("registered")

	heartbeat := time.NewTicker(time.Minute)
	defer heartbeat.Stop()

	for {
		select {
		case <-s.Context().Done():
			if err := runtime.Delete(context.Background(), server.DB); err != nil {
				logger.Error("failed to delete runtime", zap.Error(err))
				s.Exit(1)
				return
			}

			logger.Debug("deleted runtime")
			s.Exit(0)
			return
		case <-heartbeat.C:
			runtime.ExpiresAt = models.NewTime(time.Now().Add(time.Hour).UTC())

			if err := runtime.Update(s.Context(), server.DB); err != nil {
				logger.Error("failed to heartbeat runtime", zap.Error(err))
				s.Exit(1)
				return
			}

			logger.Debug("heartbeated")
		}
	}
}
