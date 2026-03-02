package server

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
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
	componentRegistry := component.NewComponentRegistry()

	// Initialize connection pool and heartbeat monitor (shared across all connections)
	pool := servicescomponent.NewConnectionPool()
	monitor := servicescomponent.NewComponentHeartbeatMonitor(componentRegistry, sessionStore, pool)
	go monitor.Start()

	// Clean up stale sockets
	for _, socket := range []string{aegisSocket, componentsSocket} {
		if err := os.RemoveAll(socket); err != nil {
			logger.Error("Failed to remove stale socket:", socket, err)
			os.Exit(1)
		}
	}

	// Aegis CLI socket
	cliListener, err := net.Listen("unix", aegisSocket)
	if err != nil {
		logger.Error("Failed to bind CLI socket:", err)
		os.Exit(1)
	}
	defer cliListener.Close()
	logger.Info("Aegis daemon listening on", aegisSocket)

	// Components socket
	componentsListener, err := net.Listen("unix", componentsSocket)
	if err != nil {
		logger.Error("Failed to bind components socket:", err)
		os.Exit(1)
	}
	defer componentsListener.Close()
	logger.Info("Components server listening on", componentsSocket)

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Accept CLI connections
	go func() {
		for {
			conn, err := cliListener.Accept()
			if err != nil {
				// Listener was closed, stop accepting
				if errors.Is(err, net.ErrClosed) {
					return
				}
				logger.Error("CLI connection error:", err)
				continue
			}
			go HandleAegis(conn, sessionStore)
		}
	}()

	// Accept component connections
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
			go servicescomponent.HandleComponentConnection(conn, componentRegistry, sessionStore, pool)
		}
	}()

	// Block until shutdown signal
	sig := <-quit
	logger.Info("Received signal, shutting down:", sig)

	// Graceful shutdown
	cliListener.Close()
	componentsListener.Close()
	logger.Info("Aegis daemon stopped")
}
