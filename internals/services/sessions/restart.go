package sessions

import (
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/nats-io/nats.go"
)

// RestartSession restarts a FINISHED session without relaunching component processes.
// The component processes are expected to still be alive and waiting — the SDK
// will handle the re-handshake when the new orchestrator sends CONFIGURE again.
func RestartSession(session *core.Session, cmd core.Command, conn net.Conn, nc *nats.Conn, tr TimeRange) error {
	if session.GetState() != core.SessionFinished {
		return fmt.Errorf("restart is only valid for FINISHED sessions (current state: %s)", session.GetState())
	}

	// Tear down old runtime if still around.
	if rt, ok := getSessionRuntime(session.ID); ok {
		if rt.orchestrator != nil {
			rt.orchestrator.Stop()
		}
		if rt.dataStream != nil {
			rt.dataStream.Stop()
		}
		clearSessionRuntime(session.ID)
	}

	// Reset session state back to INITIALIZED so SetToStarting is valid.
	if err := session.ResetToInitialized(); err != nil {
		return fmt.Errorf("restart: reset state: %w", err)
	}

	logger.Infof("Session %s: restarting (from=%d to=%d)", session.ID, tr.From, tr.To)

	// Reuse StartSession — skips LaunchComponents since entries already exist
	// but processes are alive. SetToStarting transitions INITIALIZED → STARTING.
	return StartSession(session, cmd, conn, nc, tr)
}
