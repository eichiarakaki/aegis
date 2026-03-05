package sessions

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
	"github.com/nats-io/nats.go"
)

// sessionRuntime holds runtime resources associated with a session
// that should not live in the core layer to avoid import cycles.
type sessionRuntime struct {
	orchestrator *orchestrator.Orchestrator
	dataStream   *orchestrator.DataStreamServer
}

var (
	runtimeMu       sync.RWMutex
	sessionRuntimes = make(map[string]*sessionRuntime)
)

func setSessionRuntime(sessionID string, rt *sessionRuntime) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	sessionRuntimes[sessionID] = rt
}

func getSessionRuntime(sessionID string) (*sessionRuntime, bool) {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	rt, ok := sessionRuntimes[sessionID]
	return rt, ok
}

func clearSessionRuntime(sessionID string) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	delete(sessionRuntimes, sessionID)
}

// ComponentReadyTimeout is the time StartSession waits for at least one
// component to reach CONFIGURED state before starting the orchestrator.
var ComponentReadyTimeout = 2 * time.Second

// StartSession starts the session: launches attached component binaries,
// waits for them to complete the handshake, then starts the orchestrator.
func StartSession(session *core.Session, cmd core.Command, conn net.Conn, nc *nats.Conn) error {
	if err := session.SetToStarting(); err != nil {
		return err
	}

	// Launch all binaries — stdout/stderr go to rotating log files.
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

	ds := orchestrator.NewDataStreamServer(session, nc)
	if err := ds.Start(context.Background()); err != nil {
		return fmt.Errorf("session %s: data stream server: %w", session.ID, err)
	}

	o, err := orchestrator.New(orchestrator.Config{
		SessionID: session.ID,
		Topics:    *session.Topics,
		NC:        nc,
		DS:        ds,
	})
	if err != nil {
		ds.Stop()
		return fmt.Errorf("session %s: orchestrator: %w", session.ID, err)
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

	setSessionRuntime(session.ID, &sessionRuntime{
		orchestrator: o,
		dataStream:   ds,
	})

	if err := o.Start(context.Background()); err != nil {
		ds.Stop()
		return err
	}

	return session.SetToRunning()
}

// waitForComponents polls the session registry until at least `expected`
// components reach CONFIGURED or RUNNING state, or the timeout expires.
func waitForComponents(session *core.Session, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C
		configured := session.Registry.GetByState(component.ComponentStateConfigured)
		running := session.Registry.GetByState(component.ComponentStateRunning)
		if len(configured)+len(running) >= expected {
			logger.Infof("Session %s: %d/%d component(s) ready",
				session.ID, len(configured)+len(running), expected)
			return
		}
	}

	configured := session.Registry.GetByState(component.ComponentStateConfigured)
	running := session.Registry.GetByState(component.ComponentStateRunning)
	logger.Warnf("Session %s: timeout after %s — %d/%d component(s) ready, starting orchestrator anyway",
		session.ID, timeout, len(configured)+len(running), expected)
}

// StopSession stops the session, shuts down the orchestrator and data stream.
func StopSession(session *core.Session, sessionStore *core.SessionStore) error {
	if err := session.SetToStopping(); err != nil {
		return err
	}

	if rt, ok := getSessionRuntime(session.ID); ok {
		if rt.orchestrator != nil {
			rt.orchestrator.Stop()
		}
		if rt.dataStream != nil {
			rt.dataStream.Stop()
		}
		clearSessionRuntime(session.ID)
	}

	return session.SetToStopped()
}
