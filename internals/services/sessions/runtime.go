package sessions

/* Conceptually, starting a session involves:
1. Validating that the session is in a state that can be started (e.g. Created or Stopped).
2. Transitioning the session's status to Running and recording the start time.
3. Spawning all components associated with the session.
func (m *SessionManager) StartSession(id string) error {
	session := m.GetSession(id)

	err := session.Start()
	if err != nil {
		return err
	}

	return m.spawnComponents(session)
}
*/
