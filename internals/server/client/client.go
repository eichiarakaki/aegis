package client

import (
	"encoding/json"
	"log"
	"net"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func HandleAegis(conn net.Conn) {
	defer conn.Close()

	var cmd Command
	err := json.NewDecoder(conn).Decode(&cmd)
	if err != nil {
		log.Println("Invalid command:", err)
		return
	}

	log.Printf("Received command: %s | Payload: %s\n", cmd.Type, cmd.Payload)

	switch cmd.Type {

	case "SESSION_START":
		log.Println("Starting session:", cmd.Payload)

	case "SESSION_STOP":
		log.Println("Stopping session:", cmd.Payload)

	case "SESSION_LIST":
		log.Println("Listing sessions")

	case "COMPONENT_LIST":
		log.Println("Listing components for session:", cmd.Payload)

	case "COMPONENT_GET":
		log.Println("Getting component:", cmd.Payload)

	case "COMPONENT_DESCRIBE":
		log.Println("Describing component:", cmd.Payload)

	default:
		log.Println("Unknown command")
	}
}
