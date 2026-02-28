package sessions

import (
	"fmt"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

func CreateSession(name string, mode string, sessionStore *core.SessionStore) (string, error) {
	newId, err := utils.GenerateUUID()
	if err != nil {
		return "", fmt.Errorf("couldn't create a new session: %s", err)
	}

	logger.Debug("Creating session with ID:", newId)

	s := core.NewSession(newId, name, mode)
	err = sessionStore.AddSession(s)
	if err != nil {
		return "", fmt.Errorf("couldn't add a new session to the storage: %s", err)
	}

	logger.Infof("Session created: ID=%s, Name=%s, Mode=%s", s.ID, s.Name, s.Mode)

	return newId, nil
}
