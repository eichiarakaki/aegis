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
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
	"github.com/nats-io/nats.go"
)

func HandleAegis(conn net.Conn, sessionStore *core.SessionStore, nc *nats.Conn, logStore *servicescomponent.LogStore, pool *servicescomponent.ConnectionPool) {
	defer func(conn net.Conn) {
		if err := conn.Close(); err != nil {
			logger.Error(err)
		}
	}(conn)

	var cmd core.Command
	if err := json.NewDecoder(conn).Decode(&cmd); err != nil {
		log.Println("Invalid command:", err)
		return
	}

	logger.WithRequestID(cmd.RequestID).Debugf("Received command: %s | Payload: %s", cmd.Type, cmd.Payload)

	switch cmd.Type {

	case core.CommandDaemonShutdown:
		handlers.HandleDaemonShutdown(cmd, conn, sessionStore)

	case core.CommandDaemonKill:
		handlers.HandleDaemonKill(cmd.RequestID, conn)

	// -- Session lifecycle ------------------------------------------

	case core.CommandSessionCreate:
		sessions.HandleSessionCreate(cmd, conn, sessionStore)

	case core.CommandSessionAttach:
		sessions.HandleSessionAttach(cmd, conn, sessionStore)

	case core.CommandSessionStart:
		sessions.HandleSessionStart(cmd, conn, sessionStore, nc, logStore)

	case core.CommandSessionStop:
		sessions.HandleSessionStop(cmd, conn, sessionStore)

	case core.CommandSessionList:
		sessions.HandleSessionList(cmd, conn, sessionStore)

	case core.CommandSessionState:
		sessions.HandleSessionState(cmd, conn, sessionStore)

	case core.CommandSessionDelete:
		sessions.HandleSessionDelete(cmd, conn, sessionStore)

	// -- Component inspection --------------------------------------

	case core.CommandComponentList:
		component.HandleComponentList(cmd, conn, sessionStore)

	case core.CommandComponentGet:
		component.HandleComponentGet(cmd, conn, sessionStore)

	case core.CommandComponentDescribe:
		component.HandleComponentDescribe(cmd, conn, sessionStore)

	case core.CommandComponentLogs:
		component.HandleComponentLogs(cmd, conn, sessionStore, nc, logStore)

	case core.CommandComponentLogPath:
		component.HandleComponentLogPath(cmd, conn, sessionStore)

	// -- Health ----------------------------------------------------

	case core.CommandHealthCheck:
		handlers.HandleGlobalHealth(cmd, conn, sessionStore, nc)

	case core.CommandHealthCheckSession:
		handlers.HandleHealthCheckSession(cmd, conn, sessionStore, pool)

	case core.CommandHealthCheckComp:
		handlers.HandleHealthCheckComponent(cmd, conn, sessionStore, pool)

	default:
		logger.Warn("Unknown command:", cmd.Type)
	}
}
