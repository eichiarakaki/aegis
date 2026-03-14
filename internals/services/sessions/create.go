package sessions

import (
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/utils"
)

func CreateSession(name string, mode string, market string, sessionStore *core.SessionStore) (string, error) {
	newId := utils.GenerateSessionID()

	logger.Debug("Creating session with ID:", newId)

	s := core.NewSession(newId, name, mode, market)
	err := sessionStore.AddSession(s)
	if err != nil {
		return "", fmt.Errorf("couldn't add a new session to the storage: %s", err)
	}

	logger.Infof("Session created: ID=%s, Name=%s, Mode=%s, Market=%s", s.ID, s.Name, s.Mode, s.Market)

	return newId, nil
}
