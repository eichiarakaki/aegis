package main

import (
	"github.com/eichiarakaki/aegis/internals/server"
	"github.com/eichiarakaki/aegis/internals/system"
)

func main() {
	// Print Aegis banner
	system.Print()

	// Start Aegis daemon
	server.InitDaemon()
}
