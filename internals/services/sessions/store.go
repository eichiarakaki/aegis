package sessions

import (
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
)

func GetSessionByHint(hint string, sessionStore *core.SessionStore) (*core.Session, bool) {
	logger.Debug("Session hint:", hint)

	var session *core.Session
	found := false

	_, count := sessionStore.GetSessionsByName(hint)
	if count >= 2 {
		logger.Errorf("Attempted to delete %d sessions with the same name!", count)
		return nil, false
	}
	nameSession, nameFound := sessionStore.GetSessionByName(hint)
	if nameFound {
		logger.Debug("Found session by name:", nameSession.Name)
		session = nameSession
		found = true
	}
	approxSession, approxFound := sessionStore.GetSessionByIDApproximation(hint)
	if approxFound {
		logger.Debug("Found session by ID approximation:", approxSession.ID)
		session = approxSession
		found = true
	}
	idSession, idFound := sessionStore.GetSessionByID(hint)
	if idFound {
		logger.Debug("Found session by full ID:", idSession.ID)
		session = idSession
		found = true
	}

	return session, found
}
