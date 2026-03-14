package sessions

import (
	"sync"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/orchestrator"
)

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

// ComponentReadyTimeout is how long StartSession waits for components to
// complete the full handshake (REGISTER → READY → CONFIGURE → RUNNING).
//
// 15 s gives enough headroom for:
//   - the OS to schedule the new process
//   - the Rust tokio runtime to initialise
//   - the component to connect to the Unix socket and exchange all
//     handshake messages (REGISTER, STATE_UPDATE ×2, CONFIGURE, ACK,
//     STATE_UPDATE ×2) before the orchestrator snapshots session.Topics.
//
// The previous value of 2 s was too short: if the first connection attempt
// failed for any reason the Rust SDK's default reconnect_delay (3 s) meant
// the component could never finish in time, leaving session.Topics empty and
// the orchestrator starting with no streams.
var ComponentReadyTimeout = 15 * time.Second

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
