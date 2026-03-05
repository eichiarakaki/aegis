package sessions

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/utils"
)

// LaunchComponents launches all binaries stored in session.ComponentPaths,
// injecting session credentials as environment variables so each component
// can connect and register itself without any manual configuration.
//
// Environment variables injected per process:
//
//	AEGIS_SOCKET        — path to the components Unix socket
//	AEGIS_SESSION_TOKEN — the session ID used as the registration token
//	AEGIS_COMPONENT_ID  — pre-assigned component ID for this process
//
// Processes are disowned after Start() — Aegis does not manage the OS
// process lifecycle. If a component never registers, the heartbeat monitor
// will detect its absence.
func LaunchComponents(session *core.Session) error {
	paths := session.GetComponentPaths()
	if len(paths) == 0 {
		logger.Warnf("Session %s has no attached components — skipping launch", session.ID)
		return nil
	}

	cfg, err := config.LoadGlobals()
	if err != nil {
		return fmt.Errorf("launch: load config: %w", err)
	}

	launched := 0
	for _, path := range paths {
		componentID := utils.GenerateComponentID()

		cmd := exec.Command(path)
		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("AEGIS_SOCKET=%s", cfg.ComponentsSocket),
			fmt.Sprintf("AEGIS_SESSION_TOKEN=%s", session.ID),
			fmt.Sprintf("AEGIS_COMPONENT_ID=%s", componentID),
		)
		cmd.Stdout = nil
		cmd.Stderr = nil

		if err := cmd.Start(); err != nil {
			logger.Errorf("launch: failed to start %s: %s", path, err.Error())
			continue
		}

		logger.Infof("Launched component %s (pid %d) → session %s", componentID, cmd.Process.Pid, session.ID)
		go func() { _ = cmd.Wait() }()
		launched++
	}

	if launched == 0 {
		return fmt.Errorf("launch: all %d component(s) failed to start", len(paths))
	}

	return nil
}
