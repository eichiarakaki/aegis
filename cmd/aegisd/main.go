package main

import (
	"log"
	"net"
	"os"

	"github.com/eichiarakaki/aegis/internals/config"
	client "github.com/eichiarakaki/aegis/internals/server/client"
	"github.com/eichiarakaki/aegis/internals/server/components"
)

func main() {
	log.Println("Starting Aegis daemon...")

	cfg, err := config.LoadGlobals()
	if err != nil {
		log.Fatal("Failed to load config:", err)
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

	log.Println("Aegis daemon listening on", aegisSocket)

	// Handle incoming Aegis CLI connections in a separate goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Connection error:", err)
				continue
			}
			go client.HandleAegis(conn)
		}
	}()

	// Components socket
	componentsListener, err := net.Listen("unix", componentsSocket)
	if err != nil {
		log.Fatal(err)
	}
	defer componentsListener.Close()

	log.Println("Components server listening on", componentsSocket)

	go func() {
		for {
			conn, err := componentsListener.Accept()
			if err != nil {
				log.Println("Connection error:", err)
				continue
			}
			go components.HandleComponentConnections(conn)
		}
	}()

	select {}
}
