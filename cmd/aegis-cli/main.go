package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func sendCommand(cmdType string, payload string) error {
	conn, err := net.Dial("tcp", "localhost:7000")
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
