package sessions

import (
	"fmt"
	"os"

	"github.com/eichiarakaki/aegis/internals/core"
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

// AttachComponents validates the given paths and stores them in the session.
// The binaries are NOT launched here — that happens in StartSession so the
// operator can attach components incrementally and start everything at once.
func AttachComponents(session *core.Session, paths []string) ([]string, error) {
	currentState := session.GetState()
	if currentState != core.SessionInitialized && currentState != core.SessionStopped {
		return nil, fmt.Errorf(
			"cannot attach components: session must be INITIALIZED or STOPPED, got %s",
			core.SessionStateToString(currentState),
		)
	}

	for _, path := range paths {
		if err := verifyComponent(path); err != nil {
			return nil, err
		}
	}

	for _, path := range paths {
		session.AddComponentPath(path)
	}

	return paths, nil
}
