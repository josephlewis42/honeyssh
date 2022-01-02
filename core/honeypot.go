package core

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/josephlewis42/honeyssh/commands"
	"github.com/josephlewis42/honeyssh/core/config"
	"github.com/josephlewis42/honeyssh/core/logger"
	"github.com/josephlewis42/honeyssh/core/ttylog"
	"github.com/josephlewis42/honeyssh/core/vos"
	gossh "golang.org/x/crypto/ssh"
)

type sshContextKey struct {
	name string
}

var (
	// ContextAuthPublicKey holds the public key that the client sent to the
	// server. Useful for fingerprinting.
	ContextAuthPublicKey = sshContextKey{"auth-public-key"}
	// ContextAuthPassword holds the password the client sent to the server.
	ContextAuthPassword = sshContextKey{"auth-password"}
)

type Honeypot struct {
	configuration *config.Configuration
	sharedOS      *vos.SharedOS
	toClose       listCloser
	logger        *logger.Logger
	sshServer     *ssh.Server
}

type HoneypotOpts struct {
	// Additional place to log output
	AdditionalLogger io.Writer
}

func NewHoneypot(configuration *config.Configuration, stderr io.Writer) (*Honeypot, error) {
	var toClose listCloser
	var initialized bool
	defer func() {
		if !initialized {
			toClose.Close()
		}
	}()

	// Set up the filesystem.
	vfs, err := vos.NewVFSFromConfig(configuration)
	if err != nil {
		return nil, err
	}

	// Set up the app log
	logFd, err := configuration.OpenAppLog()
	if err != nil {
		return nil, err
	}
	log.Printf("- Writing app logs to %s\n", logFd.Name())
	toClose = append(toClose, logFd)

	sharedOS := vos.NewSharedOS(vfs, commands.BuiltinProcessResolver, configuration, time.Now)
	sharedOS.SetPID(4507)

	honeypot := &Honeypot{
		configuration: configuration,
		sharedOS:      sharedOS,
		toClose:       toClose,
		logger:        logger.NewJsonLinesLogRecorder(io.MultiWriter(logFd, stderr)),
	}

	honeypot.sshServer = &ssh.Server{
		Addr: fmt.Sprintf(":%d", configuration.SSHPort),
		Handler: func(s ssh.Session) {
			honeypot.HandleConnection(s)
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			ctx.SetValue(ContextAuthPublicKey, key.Marshal())
			return false
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			ctx.SetValue(ContextAuthPassword, password)

			var successfulLogin bool
			if configuration.AllowAnyPassword {
				successfulLogin = true
			} else {
				passwords := configuration.GetPasswords(ctx.User())
				for _, allowedPass := range passwords {
					if 1 == subtle.ConstantTimeCompare([]byte(password), []byte(allowedPass)) {
						successfulLogin = true
					}
				}
			}

			// Log the login
			if !successfulLogin {
				honeypot.logger.Sessionless().Record(&logger.LogEntry_LoginAttempt{
					LoginAttempt: &logger.LoginAttempt{
						Result:     logger.OperationResult_FAILURE,
						Username:   ctx.User(),
						PublicKey:  maybeBytes(ctx.Value(ContextAuthPublicKey)),
						Password:   fmt.Sprintf("%v", password),
						RemoteAddr: fmt.Sprintf("%v", ctx.RemoteAddr()),
					},
				})
			}

			return successfulLogin
		},

		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			config := &gossh.ServerConfig{}
			config.BannerCallback = func(_ gossh.ConnMetadata) string {
				if configuration.SSHBanner != "" {
					return strings.TrimRight(configuration.SSHBanner, "\n") + "\n"
				}
				return ""
			}

			return config
		},
	}

	keyData, err := configuration.PrivateKeyPem()
	if err != nil {
		toClose.Close()
		return nil, err
	}
	honeypot.sshServer.SetOption(ssh.HostKeyPEM(keyData))

	initialized = true
	return honeypot, nil
}

func (h *Honeypot) Close() error {
	return h.toClose.Close()
}

func (h *Honeypot) HandleConnection(s ssh.Session) error {
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
	sessionLogger := h.logger.NewSession(sessionID)

	// Log panics to prevent a single connection from bringing down the whole
	// process.
	defer func() {
		if r := recover(); r != nil {
			sessionLogger.Record(&logger.LogEntry_Panic{
				Panic: &logger.Panic{
					Context:    fmt.Sprintf("Handling connection got panic: %v", r),
					Stacktrace: string(debug.Stack()),
				},
			})
		}
	}()

	// Log the login
	sessionLogger.Record(&logger.LogEntry_LoginAttempt{
		LoginAttempt: &logger.LoginAttempt{
			Result:               logger.OperationResult_SUCCESS,
			Username:             s.User(),
			PublicKey:            maybeBytes(s.Context().Value(ContextAuthPublicKey)),
			Password:             fmt.Sprintf("%s", s.Context().Value(ContextAuthPassword)),
			RemoteAddr:           fmt.Sprintf("%s", s.RemoteAddr()),
			EnvironmentVariables: s.Environ(),
			Command:              s.Command(),
			RawCommand:           s.RawCommand(),
			Subsystem:            s.Subsystem(),
		},
	})

	// Set up I/O and loging.
	logFileName := fmt.Sprintf("%s.log", time.Now().Format(time.RFC3339Nano))
	sessionLogger.Record(&logger.LogEntry_OpenTtyLog{
		OpenTtyLog: &logger.OpenTTYLog{
			Name: logFileName,
		},
	})

	logFd, err := h.configuration.CreateSessionLog(logFileName)
	if err != nil {
		return err
	}
	defer logFd.Close()

	// Start logging the terminal interactions
	vio := ttylog.NewRecorder(vos.NewVIOAdapter(s, s, s), ttylog.NewUMLLogSink(logFd))

	procName := h.configuration.OS.DefaultShell
	procArgs := []string{procName}
	if remoteCommand := s.RawCommand(); remoteCommand != "" {
		procArgs = append(procArgs, "-c", remoteCommand)
	}

	tenantOS := vos.NewTenantOS(h.sharedOS, sessionLogger, s)
	// Watch for window changes.
	{
		ptyInfo, winch, isPTY := s.Pty()
		tenantOS.SetPTY(vos.PTY{
			Width:  ptyInfo.Window.Width,
			Height: ptyInfo.Window.Height,
			Term:   ptyInfo.Term,
			IsPTY:  isPTY,
		})

		go (func() {
			for {
				select {
				case window, ok := <-winch:
					if !ok {
						return
					}
					tenantOS.SetPTY(vos.PTY{
						Width:  window.Width,
						Height: window.Height,
						Term:   ptyInfo.Term,
						IsPTY:  isPTY,
					})
				}
			}
		})()
	}

	loginProc := tenantOS.LoginProc()
	shellOS, err := loginProc.StartProcess(procName, procArgs, &vos.ProcAttr{
		Env:   append(loginProc.Environ(), s.Environ()...),
		Files: vio,
	})
	if err != nil {
		return err
	}

	// Start shell
	s.Exit(shellOS.Run())
	return nil
}

func (h *Honeypot) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", h.configuration.SSHPort)
	log.Printf("- Starting SSH server on %v\n", addr)
	h.logger.Sessionless().Record(&logger.LogEntry_HoneypotEvent{
		HoneypotEvent: &logger.HoneypotEvent{
			EventType: logger.HoneypotEvent_START,
		},
	})

	return h.sshServer.ListenAndServe()
}

func (h *Honeypot) Shutdown(ctx context.Context) error {
	defer h.Close()
	log.Printf("Terminating SSH server on %s\n", h.sshServer.Addr)
	h.logger.Sessionless().Record(&logger.LogEntry_HoneypotEvent{
		HoneypotEvent: &logger.HoneypotEvent{
			EventType: logger.HoneypotEvent_TERMINATE,
		},
	})

	return h.sshServer.Shutdown(ctx)
}

type listCloser []io.Closer

func (lc listCloser) Close() error {
	var lastErr error
	for _, v := range lc {
		if err := v.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func maybeBytes(data interface{}) []byte {
	if bytes, ok := data.([]byte); ok {
		return bytes
	}
	return nil
}
