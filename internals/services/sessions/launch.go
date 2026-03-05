package sessions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/eichiarakaki/aegis/internals/services/utils"
	"gopkg.in/natefinch/lumberjack.v2"
)

// logDir returns the directory where component log files are stored.
// $XDG_STATE_HOME/aegis/logs/<session_id>/ or ~/.local/state/aegis/logs/<session_id>/
func logDir(sessionID string) string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "aegis", "logs", sessionID)
}

// LogPath returns the log file path for a given session + component ID.
func LogPath(sessionID, componentID string) string {
	return filepath.Join(logDir(sessionID), componentID+".log")
}

// LaunchComponents launches all binaries stored in session.ComponentPaths.
// stdout and stderr of each process are written to a rotating log file at
// LogPath(sessionID, componentID). No NATS involvement — simple and robust.
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

	dir := logDir(session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("launch: create log dir %s: %w", dir, err)
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

		// Rotating log file — max 50 MB, keep last 3 files, compress old ones.
		logFile := &lumberjack.Logger{
			Filename:   LogPath(session.ID, componentID),
			MaxSize:    50, // MB
			MaxBackups: 3,
			Compress:   true,
		}

		cmd.Stdout = logFile
		cmd.Stderr = logFile

		if err := cmd.Start(); err != nil {
			logger.Errorf("launch: failed to start %s: %s", path, err)
			_ = logFile.Close()
			continue
		}

		logger.Infof("Launched component %s (pid %d) → session %s | log: %s",
			componentID, cmd.Process.Pid, session.ID, LogPath(session.ID, componentID))

		go func(lf *lumberjack.Logger) {
			_ = cmd.Wait()
			_ = lf.Close()
		}(logFile)

		launched++
	}

	if launched == 0 {
		return fmt.Errorf("launch: all %d component(s) failed to start", len(paths))
	}

	return nil
}
