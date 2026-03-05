package sessions

import (
	"context"
	"net"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
	"github.com/nats-io/nats.go"
)

// ComponentReadyTimeout is the time StartSession waits for at least one
// component to reach CONFIGURED state before starting the orchestrator.
// This gives launched binaries time to connect, register, and complete
// the handshake so their topics are populated in the session.
var ComponentReadyTimeout = 2 * time.Second

// StartSession starts the session: launches attached component binaries,
// waits for them to complete the handshake, then starts the orchestrator.
func StartSession(session *core.Session, cmd core.Command, conn net.Conn, nc *nats.Conn) error {
	if err := session.SetToStarting(); err != nil {
		return err
	}

	// Launch all binaries stored by SESSION_ATTACH.
	if err := LaunchComponents(session); err != nil {
		logger.Warnf("Session %s: component launch warning: %s", session.ID, err.Error())
	}

	paths := session.GetComponentPaths()
	if len(paths) > 0 {
		// Wait for components to connect, register, and reach CONFIGURED so
		// their topics are added to the session before the orchestrator reads them.
		logger.Infof("Session %s: waiting up to %s for components to be ready", session.ID, ComponentReadyTimeout)
		waitForComponents(session, len(paths), ComponentReadyTimeout)
	}

	if session.Topics == nil || len(*session.Topics) == 0 {
		logger.Warn("Session has no topics — orchestrator will not start")
		return session.SetToRunning()
	}

	o, err := orchestrator.New(orchestrator.Config{
		SessionID: session.ID,
		Topics:    *session.Topics,
		NC:        nc,
	})

	if err != nil {
		return err
	}

	o.OnFinished = func() {
		logger.Infof("Session %s: all data exhausted — transitioning to finished", session.ID)
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

	session.Orchestrator = o

	if err := o.Start(context.Background()); err != nil {
		return err
	}

	return session.SetToRunning()
}

// waitForComponents polls the session registry until at least `expected`
// components reach CONFIGURED state, or the timeout expires.
// Either way StartSession continues — the timeout is best-effort.
func waitForComponents(session *core.Session, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C
		configured := session.Registry.GetByState(component.ComponentStateConfigured)
		running := session.Registry.GetByState(component.ComponentStateRunning)
		ready := len(configured) + len(running)
		if ready >= expected {
			logger.Infof("Session %s: %d/%d component(s) ready", session.ID, ready, expected)
			return
		}
	}

	configured := session.Registry.GetByState(component.ComponentStateConfigured)
	running := session.Registry.GetByState(component.ComponentStateRunning)
	ready := len(configured) + len(running)
	logger.Warnf("Session %s: timeout after %s — %d/%d component(s) ready, starting orchestrator anyway",
		session.ID, timeout, ready, expected)
}

// StopSession stops the session and shuts down the orchestrator.
func StopSession(session *core.Session, sessionStore *core.SessionStore) error {
	if err := session.SetToStopping(); err != nil {
		return err
	}

	if session.Orchestrator != nil {
		session.Orchestrator.Stop()
		session.Orchestrator = nil
	}

	return session.SetToStopped()
}
