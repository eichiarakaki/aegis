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
	"github.com/eichiarakaki/aegis/internals/server/handlers/component"
	"github.com/eichiarakaki/aegis/internals/server/handlers/sessions"
)

func HandleAegis(conn net.Conn, sessionStore *core.SessionStore) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Error(err)
		}
	}(conn)

	var cmd core.Command
	err := json.NewDecoder(conn).Decode(&cmd)
	if err != nil {
		log.Println("Invalid command:", err)
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Received command: %s | Payload: %s", cmd.Type, cmd.Payload)

	switch cmd.Type {

	// -- Session lifecycle ------------------------------------------

	case "SESSION_CREATE":
		sessions.HandleSessionCreate(cmd, conn, sessionStore)

	// case "SESSION_CREATE_RUN":
	// sessions.HandleSessionCreateRun(cmd, conn, sessionStore)

	case "SESSION_ATTACH":
		sessions.HandleSessionAttach(cmd, conn, sessionStore)

	case "SESSION_START":
		sessions.HandleSessionStart(cmd, conn, sessionStore)

	case "SESSION_STOP":
		sessions.HandleSessionStop(cmd, conn, sessionStore)

	case "SESSION_LIST":
		sessions.HandleSessionList(cmd, conn, sessionStore)

	case "SESSION_STATE":
		sessions.HandleSessionState(cmd, conn, sessionStore)

	case "SESSION_DELETE":
		sessions.HandleSessionDelete(cmd, conn, sessionStore)

	// -- Component inspection --------------------------------------

	case "COMPONENT_LIST":
		component.HandleComponentList(cmd, conn, sessionStore)

	case "COMPONENT_GET":
		component.HandleComponentGet(cmd, conn, sessionStore)

	case "COMPONENT_DESCRIBE":
		component.HandleComponentDescribe(cmd, conn, sessionStore)
	// -- Health ----------------------------------------------------

	case "HEALTH_CHECK":
		handlers.HandleGlobalHealth(cmd.RequestID, conn, sessionStore)

	case "HEALTH_CHECK_SESSION":
		handlers.HandleHealthCheck(cmd.RequestID, conn, sessionStore)

	case "HEALTH_CHECK_COMPONENT":
		handlers.HandleHealthCheck(cmd.RequestID, conn, sessionStore)

	default:
		logger.Warn("Unknown command:", cmd.Type)
	}
}
