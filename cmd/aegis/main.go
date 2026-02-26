package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/eichiarakaki/aegis/internals/config"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func sendCommand(cmdType string, payload string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	socket := cfg.AegisCLISocket

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := Command{
		Type:    cmdType,
		Payload: payload,
	}

	return json.NewEncoder(conn).Encode(cmd)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: aegis-cli start|stop")
		return
	}

	switch os.Args[1] {
	case "start":
		err := sendCommand("START_SESSION", "dev")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Session started")

	case "stop":
		err := sendCommand("STOP_SESSION", "")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Session stopped")

	default:
		fmt.Println("Unknown command")
	}
}
