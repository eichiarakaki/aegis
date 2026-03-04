package sessions

import (
	"context"
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
	"github.com/nats-io/nats.go"
)

// StartSession starts the session and launches the data orchestrator.
// nc is the shared NATS connection for the server instance.
func StartSession(session *core.Session, cmd core.Command, conn net.Conn, nc *nats.Conn) error {
	if err := session.SetToStarting(); err != nil {
		return err
	}

	if len(session.Registry.GetByState(component.ComponentStateRunning)) != len(session.Registry.List()) {
		logger.Warn("Some components are not running.")
	} else {
		logger.Infof("All components ready to receive data at %s", *session.StreamSocket)
	}

	if session.Topics == nil || len(*session.Topics) == 0 {
		logger.Warn("Session has no topics — orchestrator will not start.")
		return session.SetToRunning()
	}

	o, err := orchestrator.New(orchestrator.Config{
		SessionID: session.ID,
		Topics:    *session.Topics,
		NC:        nc,
		// DataRoot is read from AEGIS_DATA_ROOT env var inside New() if not set.
	})
	if err != nil {
		return err
	}

	// Wire lifecycle callbacks so the session transitions automatically
	// when the orchestrator finishes or hits a fatal error.
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
