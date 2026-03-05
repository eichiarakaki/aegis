package handlers

import (
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/google/uuid"

	"github.com/eichiarakaki/aegis/internals/core"
)

var daemonStart = time.Now()

func HandleGlobalHealth(requestID string, conn net.Conn, store *core.SessionStore) {
	totalSessions := store.Count()
	runningSessions := store.CountByState(core.SessionRunning)

	totalComponents := store.TotalComponents()
	runningComponents := store.TotalComponentsByStateFromAllSessions(component.ComponentStateRunning)

	data := map[string]interface{}{
		"daemon": map[string]interface{}{
			"status":         "healthy",
			"uptime_seconds": int(time.Since(daemonStart).Seconds()),
			"version":        "x",
		},
		"sessions": map[string]interface{}{
			"total":   totalSessions,
			"running": runningSessions,
			"stopped": totalSessions - runningSessions,
		},
		"components": map[string]interface{}{
			"total":   totalComponents,
			"running": runningComponents,
			"failed":  totalComponents - runningComponents,
		},
	}

	core.WriteJSON(conn, core.Response{
		RequestID: requestID,
		Command:   string(core.CommandHealthCheck),
		Status:    "ok",
		Data:      data,
	})
}

func HandleSessionHealth(requestID string, conn net.Conn, store *core.SessionStore) {
	// Implement parsing real payload later
	core.WriteJSON(conn, core.Response{
		RequestID: requestID,
		Command:   string(core.CommandHealthCheckSession),
		Status:    "error",
		ErrorCode: "NOT_IMPLEMENTED",
		Message:   "Session health not implemented yet",
		Data:      map[string]interface{}{},
	})
}

func HandleComponentHealth(requestID string, conn net.Conn, store *core.SessionStore) {
	core.WriteJSON(conn, core.Response{
		RequestID: requestID,
		Command:   string(core.CommandHealthCheckComp),
		Status:    "error",
		ErrorCode: "NOT_IMPLEMENTED",
		Message:   "Component health not implemented yet",
		Data:      map[string]interface{}{},
	})
}

func HandleHealthCheck(target string, conn net.Conn, store *core.SessionStore) {
	requestID := uuid.NewString()

	switch target {

	case "", "all":
		HandleGlobalHealth(requestID, conn, store)

	case "session":
		// Expect payload: session:<id>
		HandleSessionHealth(requestID, conn, store)

	case "component":
		// Expect payload: component:<session_id>|<component_id>
		HandleComponentHealth(requestID, conn, store)

	default:
		core.WriteJSON(conn, core.Response{
			RequestID: requestID,
		Command:   string(core.CommandHealthCheck),
			Status:    "error",
			ErrorCode: "INVALID_TARGET",
			Message:   "Invalid health target",
			Data:      map[string]interface{}{},
		})
	}
}
