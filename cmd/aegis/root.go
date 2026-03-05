package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	mode  string
	paths []string
)

var rootCmd = &cobra.Command{
	Use:   "aegis",
	Short: "Aegis CLI - Control plane for Aegis daemon",
}

/////////////////////////
// DAEMON COMMANDS
/////////////////////////

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the Aegis daemon process",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Aegis daemon in the background",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadGlobals()
		if err != nil {
			log.Fatal(err)
		}

		if pid, err := readPID(cfg.AegisPIDFile); err == nil {
			if isProcessRunning(pid) {
				fmt.Printf("Aegis daemon is already running (pid %d)\n", pid)
				return
			}
			os.Remove(cfg.AegisPIDFile)
		}

		daemonBin, err := resolveDaemonBinary()
		if err != nil {
			log.Fatalf("Cannot find aegis-daemon binary: %v", err)
		}

		proc := exec.Command(daemonBin)
		proc.Stdout = nil
		proc.Stderr = nil
		proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		if err := proc.Start(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}

		if err := writePID(cfg.AegisPIDFile, proc.Process.Pid); err != nil {
			log.Printf("Warning: could not write PID file: %v", err)
		}

		fmt.Printf("Aegis daemon started (pid %d)\n", proc.Process.Pid)
	},
}

var daemonShutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Shutdown the running Aegis daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadGlobals()
		if err != nil {
			log.Fatal(err)
		}

		pid, err := readPID(cfg.AegisPIDFile)
		if err != nil {
			if stopErr := sendCommand("DAEMON_SHUTDOWN", nil); stopErr != nil {
				log.Fatalf("Daemon does not appear to be running: %v", stopErr)
			}
			return
		}

		proc, err := os.FindProcess(pid)
		if err != nil || !isProcessRunning(pid) {
			fmt.Println("Daemon is not running")
			os.Remove(cfg.AegisPIDFile)
			return
		}

		if err := proc.Signal(syscall.SIGTERM); err != nil {
			log.Fatalf("Failed to shutdown daemon (pid %d): %v", pid, err)
		}

		os.Remove(cfg.AegisPIDFile)
		fmt.Printf("Aegis daemon terminated (pid %d)\n", pid)
	},
}

var daemonKillCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kills the running Aegis daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadGlobals()
		if err != nil {
			log.Fatal(err)
		}

		pid, err := readPID(cfg.AegisPIDFile)
		if err != nil {
			if stopErr := sendCommand("DAEMON_KILL", nil); stopErr != nil {
				log.Fatalf("Daemon does not appear to be running: %v", stopErr)
			}
			return
		}

		proc, err := os.FindProcess(pid)
		if err != nil || !isProcessRunning(pid) {
			fmt.Println("Daemon is not running")
			os.Remove(cfg.AegisPIDFile)
			return
		}

		if err := proc.Signal(syscall.SIGTERM); err != nil {
			log.Fatalf("Failed to kill daemon (pid %d): %v", pid, err)
		}

		os.Remove(cfg.AegisPIDFile)
		fmt.Printf("Aegis daemon terminated (pid %d)\n", pid)
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the Aegis daemon is running",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadGlobals()
		if err != nil {
			log.Fatal(err)
		}

		pid, err := readPID(cfg.AegisPIDFile)
		if err != nil || !isProcessRunning(pid) {
			fmt.Println("Aegis daemon: stopped")
			return
		}
		fmt.Printf("Aegis daemon: running (pid %d)\n", pid)
	},
}

/////////////////////////
// SESSION COMMANDS
/////////////////////////

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
}

var sessionCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a session, optionally launching components right away",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if len(paths) == 0 {
			if err := sendCommand("SESSION_CREATE", core.SessionCreatePayload{Name: name, Mode: mode}); err != nil {
				log.Fatal(err)
			}
			return
		}
		if err := sendCommand("SESSION_CREATE_RUN", core.SessionCreateRunPayload{Name: name, Mode: mode, Paths: paths}); err != nil {
			log.Fatal(err)
		}
	},
}

var sessionAttachCmd = &cobra.Command{
	Use:   "attach <name|id>",
	Short: "Attach components to an existing session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("SESSION_ATTACH", core.SessionAttachPayload{SessionID: args[0], Paths: paths}); err != nil {
			log.Fatal(err)
		}
	},
}

var sessionStartCmd = &cobra.Command{
	Use:   "start <name|id>",
	Short: "Start a stopped session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("SESSION_START", core.SessionActionPayload{SessionID: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop <name|id>",
	Short: "Stop a running session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("SESSION_STOP", core.SessionActionPayload{SessionID: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("SESSION_LIST", nil); err != nil {
			log.Fatal(err)
		}
	},
}

var sessionStateCmd = &cobra.Command{
	Use:   "state <name|id>",
	Short: "Gets state of a specific session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("SESSION_STATE", core.SessionActionPayload{SessionID: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <name|id>",
	Short: "Delete a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("SESSION_DELETE", core.SessionActionPayload{SessionID: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// COMPONENT COMMANDS
/////////////////////////

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Inspect components",
}

var componentListCmd = &cobra.Command{
	Use:   "list <session_id>",
	Short: "List components in a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("COMPONENT_LIST", core.ComponentListPayload{SessionID: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

var componentGetCmd = &cobra.Command{
	Use:   "get <session_id> <component_id>",
	Short: "Get raw component info",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("COMPONENT_GET", core.ComponentGetPayload{SessionID: args[0], ComponentID: args[1]}); err != nil {
			log.Fatal(err)
		}
	},
}

var componentDescribeCmd = &cobra.Command{
	Use:   "describe <session_id>",
	Short: "Describe components in a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("COMPONENT_DESCRIBE", core.ComponentListPayload{SessionID: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

var (
	logFollow bool
	logAll    bool
)

var componentLogsCmd = &cobra.Command{
	Use:   "logs <session_id> <component_id|name>",
	Short: "Show logs for a component (like docker logs)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := streamComponentLogs(args[0], args[1], logFollow, logAll); err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// HEALTH COMMANDS
/////////////////////////

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Health checks for Aegis subsystems",
}

var healthCheckCmd = &cobra.Command{
	Use:   "check <target>",
	Short: "Run a health check (all|data|sessions)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("HEALTH_CHECK", core.HealthCheckPayload{Target: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

var healthSessionCmd = &cobra.Command{
	Use:   "session <id>",
	Short: "Health check for a specific session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("HEALTH_CHECK_SESSION", core.HealthCheckSessionPayload{SessionID: args[0]}); err != nil {
			log.Fatal(err)
		}
	},
}

var healthComponentCmd = &cobra.Command{
	Use:   "component <session_id> <component_id>",
	Short: "Health check for a specific component",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := sendCommand("HEALTH_CHECK_COMPONENT", core.HealthCheckComponentPayload{SessionID: args[0], ComponentID: args[1]}); err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// REQUEST / SEND
/////////////////////////

func requestJSON(cmdType string, payload interface{}) (map[string]any, error) {
	cfg, err := config.LoadGlobals()
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("unix", cfg.AegisCLISocket)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cmd := core.Command{
		RequestID: uuid.NewString(),
		Type:      cmdType,
		Payload:   payload,
	}

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		return nil, err
	}

	var response map[string]any
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, nil
	}
	return response, nil
}

func sendCommand(cmdType string, payload interface{}) error {
	resp, err := requestJSON(cmdType, payload)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	pretty, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(pretty))
	return nil
}

/////////////////////////
// LOGS
/////////////////////////

// componentLogDir returns the directory where log files are stored for a session.
// Mirrors the same logic in LaunchComponents so no daemon roundtrip is needed.
func componentLogDir(sessionID string) string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "aegis", "logs", sessionID)
}

// resolveComponentID resolves a name or ID string to a concrete component ID
// by asking the daemon for the component list of the given session.
func resolveComponentID(sessionID, ref string) (componentID, componentName string, err error) {
	resp, err := requestJSON("COMPONENT_LIST", core.ComponentListPayload{SessionID: sessionID})
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
//	all=false    → start from the current end (only new lines)
func streamComponentLogs(sessionID, ref string, follow, all bool) error {
	componentID, componentName, err := resolveComponentID(sessionID, ref)
	if err != nil {
		return err
	}

	logPath := filepath.Join(componentLogDir(sessionID), componentID+".log")

	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no log file found for component %s — has the session been started?", componentName)
		}
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	// Default: seek to end so only new lines are shown.
	// --all: start from the beginning of the file.
	if !all {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return fmt.Errorf("seek: %w", err)
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
			// EOF reached.
			if !follow {
				return nil
			}
			// --follow mode: wait for more data or Ctrl+C.
			select {
			case <-quit:
				return nil
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

/////////////////////////
// INIT
/////////////////////////

func init() {
	sessionCreateCmd.Flags().StringVar(&mode, "mode", "historical", "Session mode (realtime/historical)")
	sessionCreateCmd.Flags().StringArrayVar(&paths, "path", []string{}, "Component binary path (repeatable)")

	sessionAttachCmd.Flags().StringArrayVar(&paths, "path", []string{}, "Component binary path (repeatable)")
	_ = sessionAttachCmd.MarkFlagRequired("path")

	componentLogsCmd.Flags().BoolVarP(&logFollow, "follow", "f", false, "Follow log output (like tail -f)")
	componentLogsCmd.Flags().BoolVarP(&logAll, "all", "a", false, "Show all logs from the beginning of the file")

	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(componentCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(daemonCmd)

	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonShutdownCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonKillCmd)

	sessionCmd.AddCommand(sessionCreateCmd)
	sessionCmd.AddCommand(sessionAttachCmd)
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	sessionCmd.AddCommand(sessionStateCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)

	componentCmd.AddCommand(componentListCmd)
	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentDescribeCmd)
	componentCmd.AddCommand(componentLogsCmd)

	healthCmd.AddCommand(healthCheckCmd)
	healthCmd.AddCommand(healthSessionCmd)
	healthCmd.AddCommand(healthComponentCmd)
}
