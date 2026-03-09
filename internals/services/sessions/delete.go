package sessions

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
)

func KillProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot find process %d: %w", pid, err)
	}

	err = proc.Signal(syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("failed to send SIGKILL to pid %d: %w", pid, err)
	}

	return nil
}

// DeleteSession removes a session only is the session is stopped, finished or initialized.
func DeleteSession(session *core.Session, sessionStore *core.SessionStore) error {
	// Filtering everything else but these states
	if session.GetState() != core.SessionStopped && session.GetState() != core.SessionFinished && session.GetState() != core.SessionInitialized {
		return fmt.Errorf("can't delete session from the current state (%s). Session must be Stopped, Finished or Initialized", session.GetState())
	}

	// Terminating all component processes before deleting the session
	for _, comp := range session.Registry.List() {
		pid, err := strconv.Atoi(comp.PID)
		if err != nil {
			logger.Errorf("Failed to convert PID '%s' to int: %s", comp.PID, err.Error())
			logger.Errorf("Failed to terminate component's process: %s", err.Error())
			continue
		}
		err = KillProcess(pid)
		if err != nil {
			logger.Errorf("Failed to terminate component's process: %s", err.Error())
			continue
		}
		logger.Infof("Component terminated: %d", pid)
	}

	sessionStore.DeleteSession(session.ID)

	return nil
}
