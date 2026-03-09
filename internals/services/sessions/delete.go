package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	servicescomponent "github.com/eichiarakaki/aegis/internals/services/component"
)

// DeleteSession removes a session only if it is stopped, finished or initialized.
// Before deleting, it sends SHUTDOWN to every connected component and waits for
// a graceful exit. If a component does not respond in time, SIGKILL is used as
// a fallback.
func DeleteSession(session *core.Session, sessionStore *core.SessionStore, pool *servicescomponent.ConnectionPool) error {
	if session.GetState() != core.SessionStopped &&
		session.GetState() != core.SessionFinished &&
		session.GetState() != core.SessionInitialized {
		return fmt.Errorf(
			"can't delete session from state %s — must be Stopped, Finished or Initialized",
			session.GetState(),
		)
	}

	for _, comp := range session.Registry.List() {
		shutdownSent := false

		// Prefer a graceful shutdown over killing the process.
		if conn, ok := pool.Get(comp.ID); ok {
			env := core.NewEnvelope(
				core.MessageTypeLifecycle,
				core.CommandShutdown,
				"aegis",
				"component:"+comp.ID,
				map[string]interface{}{},
			)
			if err := json.NewEncoder(conn).Encode(env); err != nil {
				logger.Warnf("delete: failed to send SHUTDOWN to %s (%s): %v", comp.Name, comp.ID, err)
			} else {
				logger.Infof("delete: SHUTDOWN sent to %s (%s) - waiting for exit", comp.Name, comp.ID)
				shutdownSent = true
			}
		}

		// Give the component a moment to exit cleanly before resorting to SIGKILL.
		if shutdownSent {
			time.Sleep(2 * time.Second)
		}

		// Fallback: kill the process if it is still alive.
		pid, err := strconv.Atoi(comp.PID)
		if err != nil || pid == 0 {
			continue
		}
		if killErr := killProcess(pid); killErr != nil {
			logger.Warnf("delete: could not kill pid %d (%s): %v", pid, comp.Name, killErr)
		} else if !shutdownSent {
			// Only log the kill if we didn't already send SHUTDOWN.
			logger.Infof("delete: killed pid %d (%s)", pid, comp.Name)
		}
	}

	sessionStore.DeleteSession(session.ID)
	logger.Infof("delete: session %s removed", session.ID)
	return nil
}

// killProcess sends SIGKILL to a process by PID.
func killProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot find process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("SIGKILL pid %d: %w", pid, err)
	}
	return nil
}
