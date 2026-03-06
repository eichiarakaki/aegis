package sessions

import "github.com/eichiarakaki/aegis/internals/core"

// StopSession stops the session, shuts down the orchestrator and data stream.
func StopSession(session *core.Session, sessionStore *core.SessionStore) error {
	// If the orchestrator already finished and transitioned the session,
	// skip the state transition and just clean up runtime resources.
	state := session.GetState()
	if state != core.SessionRunning && state != core.SessionStarting {
		// Already stopping/stopped/finished — just tear down runtime.
		if rt, ok := getSessionRuntime(session.ID); ok {
			if rt.orchestrator != nil {
				rt.orchestrator.Stop()
			}
			if rt.dataStream != nil {
				rt.dataStream.Stop()
			}
			clearSessionRuntime(session.ID)
		}
		return nil
	}

	if err := session.SetToStopping(); err != nil {
		return err
	}

	return nil
}
