package server

import (
	"log"
	"net"
	"os"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/logger"
	components "github.com/eichiarakaki/aegis/internals/services/component"
)

func InitDaemon() {
	// Initialize Daemon
	logger.Info("Starting Aegis daemon...")

	cfg, err := config.LoadGlobals()
	if err != nil {
		logger.Error("Failed to load config:", err)
	}

	aegisSocket := cfg.AegisCLISocket
	componentsSocket := cfg.ComponentsSocket

	// Remove old sockets
	os.RemoveAll(aegisSocket)
	os.RemoveAll(componentsSocket)

	// Aegis CLI socket
	listener, err := net.Listen("unix", aegisSocket)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	logger.Info("Aegis daemon listening on", aegisSocket)

	// Handle incoming Aegis CLI connections in a separate goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.Error("Connection error:", err)
				continue
			}
			go HandleAegis(conn)
		}
	}()

	// Components socket
	componentsListener, err := net.Listen("unix", componentsSocket)
	if err != nil {
		log.Fatal(err)
	}
	defer componentsListener.Close()

	logger.Info("Components server listening on", componentsSocket)

	go func() {
		for {
			conn, err := componentsListener.Accept()
			if err != nil {
				logger.Error("Connection error:", err)
				continue
			}
			go components.HandleComponentConnections(conn)
		}
	}()

	select {}
}
