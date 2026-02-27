/*
This file decodes incoming commands from the client and routes them to the appropriate handlers.
*/
package server

import (
	"encoding/json"
	"log"
	"net"

	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/server/handlers"
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

	logger.Infof("Received command: %s | Payload: %s", cmd.Type, cmd.Payload)

	switch cmd.Type {

	case "SESSION_START":
		logger.Info("Starting session:", cmd.Payload)

	case "SESSION_STOP":
		logger.Info("Stopping session:", cmd.Payload)

	case "SESSION_LIST":
		logger.Info("Listing sessions")

	case "COMPONENT_LIST":
		logger.Info("Listing components for session:", cmd.Payload)

	case "COMPONENT_GET":
		logger.Info("Getting component:", cmd.Payload)

	case "COMPONENT_DESCRIBE":
		logger.Info("Describing component:", cmd.Payload)

	case "HEALTH_CHECK":
		handlers.HandleHealthCheck(cmd.Payload, conn)

	default:
		logger.Warn("Unknown command:", cmd.Type)
	}
}
