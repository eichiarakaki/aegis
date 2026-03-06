package sessions

import (
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
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

// waitForComponents polls the session registry until at least `expected`
// components reach CONFIGURED or RUNNING state, or the timeout expires.
func waitForComponents(session *core.Session, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C
		configured := session.Registry.GetByState(core.ComponentStateConfigured)
		running := session.Registry.GetByState(core.ComponentStateRunning)
		if len(configured)+len(running) >= expected {
			logger.Infof("Session %s: %d/%d component(s) ready",
				session.ID, len(configured)+len(running), expected)
			return
		}
	}

	configured := session.Registry.GetByState(core.ComponentStateConfigured)
	running := session.Registry.GetByState(core.ComponentStateRunning)
	logger.Warnf("Session %s: timeout after %s — %d/%d component(s) ready, starting orchestrator anyway",
		session.ID, timeout, len(configured)+len(running), expected)
}
