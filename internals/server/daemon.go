package server

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/eichiarakaki/aegis/internals/services/component/manager"
	"github.com/nats-io/nats.go"
)

func InitDaemon() {
	logger.Info("Starting Aegis daemon...")

	cfg, err := config.LoadGlobals()
	if err != nil {
		logger.Error("Failed to load config:", err)
		os.Exit(1)
	}

	aegisSocket := cfg.AegisCLISocket
	componentsSocket := cfg.ComponentsSocket

	sessionStore := core.NewSessionStore()

	pool := servicescomponent.NewConnectionPool()
	monitor := servicescomponent.NewComponentHeartbeatMonitor(sessionStore, pool)
	go monitor.Start()

	// LogStore keeps the last 500 lines per component for backlog replay.
	logStore := servicescomponent.NewLogStore(500)

	for _, socket := range []string{aegisSocket, componentsSocket} {
		if err := os.RemoveAll(socket); err != nil {
			logger.Error("Failed to remove stale socket:", socket, err)
			os.Exit(1)
		}
	}

	cliListener, err := net.Listen("unix", aegisSocket)
	if err != nil {
		logger.Error("Failed to bind CLI socket:", err)
		os.Exit(1)
	}
	defer func(cliListener net.Listener) {
		err := cliListener.Close()
		if err != nil {
			logger.Error("Failed to close CLI socket:", err)
		}
	}(cliListener)
	logger.Info("Aegis daemon listening on", aegisSocket)

	componentsListener, err := net.Listen("unix", componentsSocket)
	if err != nil {
		logger.Error("Failed to bind components socket:", err)
		os.Exit(1)
	}
	defer func(componentsListener net.Listener) {
		err := componentsListener.Close()
		if err != nil {
			logger.Error("Failed to close components socket:", err)
		}
	}(componentsListener)
	logger.Info("Components server listening on", componentsSocket)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		logger.Error(err)
		return
	}
	defer nc.Close()

	go func() {
		for {
			conn, err := cliListener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				logger.Error("CLI connection error:", err)
				continue
			}
			go HandleAegis(conn, sessionStore, nc, logStore)
		}
	}()

	go func() {
		for {
			conn, err := componentsListener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				logger.Error("Component connection error:", err)
				continue
			}
			go manager.HandleComponentConnection(conn, sessionStore, pool)
		}
	}()

	sig := <-quit
	logger.Info("Received signal, shutting down:", sig)

	err = cliListener.Close()
	if err != nil {
		logger.Error("Failed to close CLI socket:", err)
		return
	}
	err = componentsListener.Close()
	if err != nil {
		logger.Error("Failed to close components socket:", err)
		return
	}
	logger.Info("Aegis daemon stopped")
}
