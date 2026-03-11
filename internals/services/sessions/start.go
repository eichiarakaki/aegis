package sessions

import (
	"context"
	"fmt"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
	"github.com/nats-io/nats.go"
)

// TimeRange optionally restricts historical playback to [From, To] (unix ms).
// Zero values mean no bound. Ignored when session.Mode == "realtime".
type TimeRange struct {
	From int64
	To   int64
}

// StartSession starts the session: launches attached component binaries,
// waits for them to complete the handshake, then starts the orchestrator.
// On any failure after SetToStarting the session is rolled back to INITIALIZED.
func StartSession(session *core.Session, cmd core.Command, conn net.Conn, nc *nats.Conn, tr TimeRange) error {
	if err := session.SetToStarting(); err != nil {
		return err
	}

	rollback := func(cause error) error {
		session.ForceState(core.SessionInitialized)
		return cause
	}

	if err := LaunchComponents(session); err != nil {
		logger.Warnf("Session %s: component launch warning: %s", session.ID, err.Error())
	}

	paths := session.GetComponentPaths()
	registered := session.Registry.Count()
	expected := len(paths)
	if registered > expected {
		expected = registered
	}
	if expected > 0 {
		logger.Infof("Session %s: waiting up to %s for %d component(s) to be ready",
			session.ID, ComponentReadyTimeout, expected)
		waitForComponents(session, expected, ComponentReadyTimeout)
	}

	if session.Topics == nil || len(*session.Topics) == 0 {
		logger.Warn("Session has no topics — orchestrator will not start")
		return session.SetToRunning()
	}

	logger.Infof("Session %s: mode=%s", session.ID, session.Mode)

	ds := orchestrator.NewDataStreamServer(session, nc)
	if err := ds.Start(context.Background()); err != nil {
		return rollback(fmt.Errorf("session %s: data stream server: %w", session.ID, err))
	}

	o, err := orchestrator.New(orchestrator.Config{
		SessionID: session.ID,
		Topics:    *session.Topics,
		NC:        nc,
		DS:        ds,
		Mode:      session.Mode,
		FromTS:    tr.From,
		ToTS:      tr.To,
	})
	if err != nil {
		ds.Stop()
		return rollback(fmt.Errorf("session %s: orchestrator: %w", session.ID, err))
	}

	// OnFinished is only meaningful in historical mode.
	o.OnFinished = func() {
		logger.Infof("Session %s: all data exhausted - transitioning to finished", session.ID)
		if err := session.SetToStopping(); err != nil {
			logger.Errorf("Session %s: SetToStopping: %s", session.ID, err.Error())
			return
		}
		if err := session.SetToStopped(); err != nil {
			logger.Errorf("Session %s: SetToStopped: %s", session.ID, err.Error())
			return
		}
		if err := session.SetToFinished(); err != nil {
			logger.Errorf("Session %s: SetToFinished: %s", session.ID, err.Error())
		}
	}
	o.OnError = func(err error) {
		logger.Errorf("Session %s: orchestrator fatal error: %s", session.ID, err.Error())
		_ = session.SetToError()
	}

	setSessionRuntime(session.ID, &sessionRuntime{
		orchestrator: o,
		dataStream:   ds,
	})

	if err := o.Start(context.Background()); err != nil {
		ds.Stop()
		return rollback(err)
	}

	return session.SetToRunning()
}
