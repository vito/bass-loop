package runnel

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/gliderlabs/ssh"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

type SSHHandler struct {
}

const (
	ForwardedStreamlocalChannelType     = "forwarded-streamlocal@openssh.com"
	StreamlocalForwardChannelType       = "streamlocal-forward@openssh.com"
	CancelStreamlocalForwardChannelType = "cancel-streamlocal-forward@openssh.com"
)

type ForwardHandler struct {
	DB *sql.DB

	processCtx context.Context

	forwards map[string]net.Listener
	sync.Mutex
}

func NewForwardHandler(ctx context.Context, db *sql.DB) *ForwardHandler {
	return &ForwardHandler{
		DB: db,

		processCtx: ctx,
		forwards:   make(map[string]net.Listener),
	}
}

// streamLocalChannelForwardMsg is a struct used for SSH2_MSG_GLOBAL_REQUEST message
// with "streamlocal-forward@openssh.com"/"cancel-streamlocal-forward@openssh.com" string.
type streamLocalChannelForwardMsg struct {
	socketPath string
}

func (h *ForwardHandler) HandleStreamlocalForward(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	logger := zapctx.FromContext(h.processCtx).With(zap.String("request", req.Type))

	var reqPayload streamlocalChannelForwardMsg
	if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
		logger.Error("malformed request", zap.Error(err))
		return false, []byte(err.Error())
	}

	sessionID := ctx.SessionID()
	logicalSocketPath := reqPayload.SocketPath

	logger.Info("handling streamlocal-forward",
		zap.String("socket", logicalSocketPath))

	svcName := filepath.Base(logicalSocketPath)

	realSocketPath, err := xdg.StateFile(path.Join(
		"bass-loop",
		"svc",
		ctx.User(),
		sessionID[:16], // avoid exceeding unix socket path max length (~108)
		svcName+".sock",
	))
	if err != nil {
		logger.Error("failed to create socket", zap.Error(err))
		return false, []byte(err.Error())
	}

	userIDVal := ctx.Value(userIdKey{})
	if userIDVal == nil {
		logger.Error("no user ID in context", zap.Error(err))
		return false, []byte("user id not found in context - this should never happen")
	}

	userID := userIDVal.(string)

	svc := models.Service{
		UserID:      userID,
		RuntimeName: sessionID,
		Service:     svcName,
		Addr:        (&url.URL{Scheme: "unix", Path: realSocketPath}).String(),
	}

	if err := svc.Upsert(ctx, h.DB); err != nil {
		logger.Error("failed to upsert service", zap.Error(err))
		return false, []byte("user id not found in context - this should never happen")
	}

	logger = logger.With(zap.String("service", svc.Service))

	ln, err := net.Listen("unix", realSocketPath)
	if err != nil {
		logger.Error("failed to listen", zap.Error(err))
		return false, []byte(err.Error())
	}

	h.trackListener(sessionID, logicalSocketPath, ln)

	eg := new(errgroup.Group)
	eg.Go(func() error {
		defer h.closeListener(sessionID, logicalSocketPath)
		<-ctx.Done()
		return ctx.Err()
	})
	eg.Go(func() error {
		defer h.closeListener(sessionID, logicalSocketPath)
		return h.listen(ctx, ln, logicalSocketPath)
	})

	go func() {
		if err := eg.Wait(); err != nil &&
			!errors.Is(err, context.Canceled) &&
			!strings.HasSuffix(err.Error(), "use of closed network connection") {
			logger.Error("error forwarding", zap.Error(err))
		} else {
			logger.Debug("completed forwarding")
		}

		if err := svc.Delete(context.Background(), h.DB); err != nil {
			logger.Error("failed to delete service", zap.Error(err))
		} else {
			logger.Debug("deleted service")
		}
	}()

	return true, nil
}

func (h *ForwardHandler) HandleCancelStreamlocalForward(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	logger := zapctx.FromContext(h.processCtx).With(zap.String("request", req.Type))

	logger.Info("handling cancel-streamlocal-forward")

	var reqPayload streamlocalChannelForwardMsg
	if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
		logger.Error("malformed request", zap.Error(err))
		return false, []byte(err.Error())
	}

	h.closeListener(ctx.SessionID(), reqPayload.SocketPath)

	return true, nil
}

func (h *ForwardHandler) listen(ctx ssh.Context, ln net.Listener, logicalSocketPath string) error {
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)

	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}

		logger := zapctx.FromContext(h.processCtx).With(zap.String("conn", c.RemoteAddr().String()))

		go func() {
			payload := gossh.Marshal(&forwardedStreamlocalPayload{
				SocketPath: logicalSocketPath,
			})

			ch, reqs, err := conn.OpenChannel(ForwardedStreamlocalChannelType, payload)
			if err != nil {
				logger.Error("failed to open channel", zap.Error(err))
				c.Close()
				return
			}

			closeAll := func() {
				ch.Close()
				c.Close()
			}

			eg := new(errgroup.Group)
			eg.Go(func() error {
				defer closeAll()
				gossh.DiscardRequests(reqs)
				return nil
			})
			eg.Go(func() error {
				defer closeAll()
				_, err := io.Copy(ch, c)
				return err
			})
			eg.Go(func() error {
				defer closeAll()
				_, err := io.Copy(c, ch)
				return err
			})

			if err := eg.Wait(); err != nil {
				logger.Error("encountered error while forwarding", zap.Error(err))
			}
		}()
	}
}

func (h *ForwardHandler) trackListener(sessionID, socketPath string, ln net.Listener) {
	id := sessionID + ":" + socketPath

	h.Lock()
	h.forwards[id] = ln
	h.Unlock()
}

func (h *ForwardHandler) closeListener(sessionID, socketPath string) {
	id := sessionID + ":" + socketPath

	h.Lock()
	defer h.Unlock()

	ln, ok := h.forwards[id]
	if ok {
		ln.Close()
	}

	delete(h.forwards, id)
}

type streamlocalChannelForwardMsg struct {
	SocketPath string
}

type forwardedStreamlocalPayload struct {
	SocketPath string
	Reserved0  string
}
