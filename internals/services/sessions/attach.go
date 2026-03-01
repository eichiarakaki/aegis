package sessions

import (
	"fmt"
	"os"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/services/sessions/utils"
)

// verifyComponent validates if the executable exists and is executable.
func verifyComponent(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("executable not found: %s", path)
		}
		return fmt.Errorf("failed to access executable: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not an executable: %s", path)
	}

	if (info.Mode() & 0111) == 0 {
		return fmt.Errorf("file is not executable: %s", path)
	}

	return nil
}

// AttachComponents attaches components to a STOPPED/INITIALIZED session.
func AttachComponents(session *core.Session, paths []string) ([]core.Component, error) {

	// We first check the current state of the session the user tryna inject components.
	currentState := session.GetState()
	if currentState != core.SessionInitialized && currentState != core.SessionStopped {
		return nil, fmt.Errorf("session is not initialized or stopped: %s", core.SessionStateToString(session.GetState()))
	}

	// Verifies component's path and type (it must be an executable)
	var validComponents []string
	var invalidComponents []string
	for _, path := range paths {
		err := verifyComponent(path)
		if err != nil {
			invalidComponents = append(invalidComponents, path)
			continue
		}
		validComponents = append(validComponents, path)
	}

	// Building components
	var components []core.Component
	for _, component := range validComponents {
		newID, err := utils.GenerateUUID()
		if err != nil {
			return nil, err
		}

		// The other areas will be filled when the component opens a connection with Aegis
		components = append(components, core.Component{
			ID:    &newID,
			State: core.ComponentPending,
			Path:  component,
		})
	}

	return components, nil
}
