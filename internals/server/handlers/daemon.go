package handlers

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/eichiarakaki/aegis/internals/core"
)

func HandleDaemonShutdown(cmd core.Command, conn net.Conn, sessionStore *core.SessionStore) {

	// TODO: Make a gracefully shutdown

	// Finally we can shutdown the process
	err := syscall.Kill(os.Getpid(), syscall.SIGTERM)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: cmd.RequestID,
			Command:   core.CommandDaemonShutdown,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Error killing the process: %s", err.Error()),
			Data:      nil,
		})
	}
}

func HandleDaemonKill(requestID string, conn net.Conn) {
	// Killing the process
	err := syscall.Kill(os.Getpid(), syscall.SIGTERM)
	if err != nil {
		core.WriteJSON(conn, core.Response{
			RequestID: requestID,
			Command:   core.CommandDaemonKill,
			Status:    core.ERROR,
			Message:   fmt.Sprintf("Error killing the process: %s", err.Error()),
			Data:      nil,
		})
		return
	}

	core.WriteJSON(conn, core.Response{
		RequestID: requestID,
		Command:   core.CommandDaemonKill,
		Status:    core.OK,
		Data:      nil,
	})
}
