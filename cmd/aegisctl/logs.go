package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
)

// componentLogDir returns the directory where log files are stored for a
// session. Mirrors the same path logic used inside the daemon so no extra
// round-trip is required.
func componentLogDir(sessionID string) string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "aegis", "logs", sessionID)
}

// resolveComponentID resolves a component name or ID string to its canonical
// (ID, name) pair by querying the daemon.
func resolveComponentID(sessionID, ref string) (id, name string, err error) {
	resp, err := requestJSON(core.CommandComponentList, core.ComponentListPayload{SessionID: sessionID})
	if err != nil {
		return "", "", fmt.Errorf("list components: %w", err)
	}
	if resp == nil {
		return "", "", errors.New("daemon returned no response")
	}

	dataBytes, err := json.Marshal(resp["data"])
	if err != nil {
		return "", "", fmt.Errorf("marshal data: %w", err)
	}

	var list struct {
		Components []struct {
			ID   string `json:"ID"`
			Name string `json:"Name"`
		} `json:"components"`
	}
	if err := json.Unmarshal(dataBytes, &list); err != nil {
		return "", "", fmt.Errorf("decode components: %w", err)
	}

	for _, c := range list.Components {
		if c.ID == ref || c.Name == ref {
			return c.ID, c.Name, nil
		}
	}
	return "", "", fmt.Errorf("component %q not found in session %s", ref, sessionID)
}

// streamComponentLogs tails the log file for the given component.
//
//	follow=true  → keep streaming new lines (like docker logs -f)
//	follow=false → print existing content and exit
//	all=true     → start from the beginning of the file
//	all=false    → start from the current end (only new lines when following)
func streamComponentLogs(sessionID, ref string, follow, all bool) error {
	componentID, componentName, err := resolveComponentID(sessionID, ref)
	if err != nil {
		return err
	}

	logPath := filepath.Join(componentLogDir(sessionID), componentID+".log")

	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no log file found for component %q — has the session been started?", componentName)
		}
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	if !all {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return fmt.Errorf("seek to end: %w", err)
		}
	}

	fmt.Printf("Logs — component: %s  id: %s\n", componentName, componentID)
	fmt.Printf("File: %s\n\n", logPath)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Print(line)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			if !follow {
				return nil
			}
			select {
			case <-quit:
				return nil
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}
