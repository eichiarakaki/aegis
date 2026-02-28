/*
Package server
This file decodes incoming commands from the client and routes them to the appropriate handlers.
*/
package server

import (
	"encoding/json"
	"log"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/server/handlers"
	"github.com/eichiarakaki/aegis/internals/server/handlers/sessions"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func HandleAegis(conn net.Conn, sessionStore *core.SessionStore) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Error(err)
		}
	}(conn)

	var cmd Command
	err := json.NewDecoder(conn).Decode(&cmd)
	if err != nil {
		log.Println("Invalid command:", err)
		return
	}

	logger.Infof("Received command: %s | Payload: %s", cmd.Type, cmd.Payload)

	switch cmd.Type {

	// -- Session lifecycle ------------------------------------------

	case "SESSION_CREATE":
		sessions.HandleSessionCreate(cmd.Payload, conn, sessionStore)

	case "SESSION_CREATE_RUN":
		sessions.HandleSessionCreateRun(cmd.Payload, conn, sessionStore)

	case "SESSION_ATTACH":
		sessions.HandleSessionAttach(cmd.Payload, conn, sessionStore)

	case "SESSION_START":
		sessions.HandleSessionStart(cmd.Payload, conn, sessionStore)

	case "SESSION_STOP":
		sessions.HandleSessionStop(cmd.Payload, conn, sessionStore)

	case "SESSION_LIST":
		sessions.HandleSessionList(conn, sessionStore)

	case "SESSION_STATUS":
		sessions.HandleSessionStatus(conn, cmd.Payload, sessionStore)

	case "SESSION_DELETE":
		sessions.HandleSessionDelete(cmd.Payload, conn, sessionStore)

	// -- Component inspection --------------------------------------

	case "COMPONENT_LIST":
		logger.Info("Listing components for session:", cmd.Payload, sessionStore)

	case "COMPONENT_GET":
		logger.Info("Getting component:", cmd.Payload, sessionStore)

	case "COMPONENT_DESCRIBE":
		logger.Info("Describing component:", cmd.Payload, sessionStore)

	// -- Health ----------------------------------------------------

	case "HEALTH_CHECK":
		handlers.HandleHealthCheck(cmd.Payload, conn, sessionStore)

	default:
		logger.Warn("Unknown command:", cmd.Type)
	}
}
