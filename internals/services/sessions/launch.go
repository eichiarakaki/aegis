package sessions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/utils"
	"gopkg.in/natefinch/lumberjack.v2"
)

func logDir(sessionID string) string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "aegis", "logs", sessionID)
}

func LogPath(sessionID, componentID string) string {
	return filepath.Join(logDir(sessionID), componentID+".log")
}

// LaunchComponents launches all binaries in session.ComponentEntries.
// If a component already has a live entry in the registry (e.g. after a restart),
// its binary is not relaunched — the existing process is expected to still be running.
func LaunchComponents(session *core.Session) error {
	entries := session.GetComponentEntries()
	if len(entries) == 0 {
		logger.Warnf("Session %s has no attached components — skipping launch", session.ID)
		return nil
	}

	cfg, err := config.LoadGlobals()
	if err != nil {
		return fmt.Errorf("launch: load config: %w", err)
	}

	dir := logDir(session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("launch: create log dir %s: %w", dir, err)
	}

	launched := 0
	skipped := 0
	for _, entry := range entries {
		componentID := entry.ComponentID
		if componentID == "" {
			componentID = utils.GenerateComponentID()
			logger.Warnf("launch: no pre-assigned ID for %s — generated %s", entry.Path, componentID)
		}

		// Skip relaunching only if the process has already connected and completed
		// at least part of the handshake. INIT and REGISTERED are placeholder states
		// that exist before the process connects — they must be launched.
		comp, exists := session.Registry.Get(componentID)
		if exists {
			switch comp.State {
			case core.ComponentStateInitializing,
				core.ComponentStateReady,
				core.ComponentStateConfigured,
				core.ComponentStateRunning,
				core.ComponentStateWaiting:
				logger.Infof("launch: component %s (%s) already live (state=%s) — skipping relaunch",
					comp.Name, componentID, comp.State)
				skipped++
				continue
			}
		}

		cmd := exec.Command(entry.Path)
		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("AEGIS_SOCKET=%s", cfg.ComponentsSocket),
			fmt.Sprintf("AEGIS_SESSION_TOKEN=%s", session.ID),
			fmt.Sprintf("AEGIS_COMPONENT_ID=%s", componentID),
		)

		logFile := &lumberjack.Logger{
			Filename:   LogPath(session.ID, componentID),
			MaxSize:    50,
			MaxBackups: 3,
			Compress:   true,
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile

		if err := cmd.Start(); err != nil {
			logger.Errorf("launch: failed to start %s: %s", entry.Path, err)
			_ = logFile.Close()
			continue
		}

		logger.Infof("Launched %s (pid %d, id %s) -> session %s",
			entry.Path, cmd.Process.Pid, componentID, session.ID)
		// Registering component's PID
		comp.PID = strconv.Itoa(cmd.Process.Pid)

		go func(lf *lumberjack.Logger) {
			_ = cmd.Wait()
			_ = lf.Close()
		}(logFile)

		launched++
	}

	if launched == 0 && skipped == 0 {
		return fmt.Errorf("launch: all %d component(s) failed to start", len(entries))
	}
	return nil
}
