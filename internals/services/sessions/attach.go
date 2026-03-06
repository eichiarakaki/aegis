package sessions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/services/utils"
)

// verifyComponent validates that the path exists and is executable.
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

// AttachComponents validates paths, stores them in the session, and registers
// a placeholder Component for each one. The placeholder holds State=INIT so
// the CLI can see it immediately. When the binary connects and sends REGISTER,
// HandleComponentConnection finds the existing entry by ID and fills in the
// real capabilities, name, and version.
func AttachComponents(session *core.Session, paths []string) ([]core.ComponentRef, error) {
	currentState := session.GetState()
	if currentState != core.SessionInitialized && currentState != core.SessionStopped {
		return nil, fmt.Errorf(
			"cannot attach components: session must be INITIALIZED or STOPPED, got %s",
			string(currentState),
		)
	}

	for _, path := range paths {
		if err := verifyComponent(path); err != nil {
			return nil, err
		}
	}

	refs := make([]core.ComponentRef, 0, len(paths))
	for _, path := range paths {
		session.AddComponentPath(path)

		componentID := utils.GenerateComponentID()
		name := filepath.Base(path)

		comp := &core.Component{
			ID:        componentID,
			SessionID: session.ID,
			Name:      name,
			State:     core.ComponentStateInit,
		}

		if err := session.Registry.Register(comp); err != nil {
			return nil, fmt.Errorf("failed to register placeholder for %s: %w", path, err)
		}

		// Store the pre-assigned ID alongside the path so LaunchComponents
		// can pass it via AEGIS_COMPONENT_ID.
		session.AddComponentIDForPath(path, componentID)

		refs = append(refs, core.ComponentRef{
			Name:  name,
			State: string(core.ComponentStateInit),
		})
	}

	return refs, nil
}
