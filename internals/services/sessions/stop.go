package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// StopSession stops the session synchronously and always results in STOPPED,
// even if OnFinished raced ahead and transitioned to FINISHED first.
func StopSession(session *core.Session, sessionStore *core.SessionStore) error {
	// Tear down runtime unconditionally.
	if rt, ok := getSessionRuntime(session.ID); ok {
		if rt.orchestrator != nil {
			rt.orchestrator.Stop()
		}
		if rt.dataStream != nil {
			rt.dataStream.Stop()
		}
		clearSessionRuntime(session.ID)
	}

	switch session.GetState() {
	case core.SessionStopped:
		return nil

	case core.SessionFinished:
		// OnFinished raced ahead — force back to STOPPED so resume works.
		session.ForceState(core.SessionStopped)
		logger.Infof("Session %s: stopped (was finished)", session.ID)
		return nil

	case core.SessionStopping:
		// Mid-transition from OnFinished — complete to STOPPED.
		_ = session.SetToStopped()
		return nil

	case core.SessionRunning, core.SessionStarting:
		if err := session.SetToStopping(); err != nil {
			return err
		}
		if err := session.SetToStopped(); err != nil {
			return err
		}
		logger.Infof("Session %s: stopped", session.ID)
		return nil

	default:
		return nil
	}
}
