package sessions

import (
	"net"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/core/component"
	"github.com/eichiarakaki/aegis/internals/logger"
)

// StartSession starts a session only if the given session is 'SessionStarting' OR 'SessionStopped'
func StartSession(session *core.Session, cmd core.Command, conn net.Conn) error {

	err := session.SetToStarting()
	if err != nil {
		return err
	}

	if len(session.Registry.GetByState(component.ComponentStateRunning)) != len(session.Registry.List()) {
		logger.Warn("Some components are not running.")
	} else {
		logger.Infof("All components ready to receive data at %s", *session.StreamSocket)
	}

	// TODO: Here you should call a function to actually start running the session.

	err = session.SetToRunning()
	if err != nil {
		return err
	}

	return nil
}

func StopSession(session *core.Session, sessionStore *core.SessionStore) error {

	err := session.SetToStopping()
	if err != nil {
		return err
	}

	// TODO: Stop the session ONLY IF ALL THE COMPONENTS are stopped.

	err = session.SetToStopped()

	if err != nil {
		return err
	}

	// TODO: idk you can do something here

	return nil
}
