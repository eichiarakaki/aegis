package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/spf13/cobra"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

var (
	mode  string
	paths []string

	session string
)

var rootCmd = &cobra.Command{
	Use:   "aegis",
	Short: "Aegis CLI - Control plane for Aegis daemon",
}

/////////////////////////
// SESSION COMMANDS
/////////////////////////

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
}

// aegis session create <name> --mode <mode> [--path <p>...]
//
// Without --path: creates the session and returns.
// With    --path: creates the session and immediately spawns the components.
var sessionCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a session, optionally launching components right away",
	Example: `  aegis session create my-session --mode live
  aegis session create my-session --mode backtest --path ./comp1 --path ./comp2`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if len(paths) == 0 {
			// Plain create — no components to run yet.
			err := sendCommand("SESSION_CREATE", fmt.Sprintf("%s|%s", name, mode))
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		// Create + launch components in one shot.
		err := sendCommand("SESSION_CREATE_RUN", buildRunPayload(name, mode, paths))
		if err != nil {
			log.Fatal(err)
		}
	},
}

// aegis session attach <name|id> --path <p>...
//
// Attaches new components to an already existing session.
var sessionAttachCmd = &cobra.Command{
	Use:     "attach <name|id>",
	Short:   "Attach components to an existing session",
	Example: `  aegis session attach my-session --path ./comp1 --path ./comp2`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nameOrID := args[0]
		err := sendCommand("SESSION_ATTACH", buildRunPayload(nameOrID, "", paths))
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionStartCmd = &cobra.Command{
	Use:   "start <name|id>",
	Short: "Start a stopped session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_START", args[0])
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop <name|id>",
	Short: "Stop a running session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_STOP", args[0])
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_LIST", "")
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionStatusCmd = &cobra.Command{
	Use:   "status <name|id>",
	Short: "Gets status of a specific session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_STATUS", args[0])
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <name|id>",
	Short: "Delete a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_DELETE", args[0])
		if err != nil {
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
	Use:   "list",
	Short: "List components in a session",
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("COMPONENT_LIST", session)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var componentGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get raw component info",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("COMPONENT_GET", fmt.Sprintf("%s|%s", session, args[0]))
		if err != nil {
			log.Fatal(err)
		}
	},
}

var componentDescribeCmd = &cobra.Command{
	Use:   "describe <id>",
	Short: "Describe a component (formatted output)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("COMPONENT_DESCRIBE", fmt.Sprintf("%s|%s", session, args[0]))
		if err != nil {
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
		err := sendCommand("HEALTH_CHECK", args[0])
		if err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// HELPERS
/////////////////////////

// buildRunPayload encodes session identity, mode, and component paths
// into a single pipe-delimited string:
//
//	<name_or_id>|<mode>|<path1>,<path2>,...
func buildRunPayload(nameOrID, mode string, paths []string) string {
	return fmt.Sprintf("%s|%s|%s", nameOrID, mode, strings.Join(paths, ","))
}

/////////////////////////
// SEND
/////////////////////////

func sendCommand(cmdType, payload string) error {
	cfg, err := config.LoadGlobals()
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", cfg.AegisCLISocket)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(Command{Type: cmdType, Payload: payload}); err != nil {
		return err
	}

	var response map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&response); err == nil {
		pretty, _ := json.MarshalIndent(response, "", "  ")
		fmt.Println(string(pretty))
	}

	return nil
}

/////////////////////////
// INIT
/////////////////////////

func init() {
	// session create flags
	sessionCreateCmd.Flags().StringVar(&mode, "mode", "realtime", "Session mode (realtime/historical)")
	sessionCreateCmd.Flags().StringArrayVar(&paths, "path", []string{}, "Component binary path (repeatable)")

	// session attach flags
	sessionAttachCmd.Flags().StringArrayVar(&paths, "path", []string{}, "Component binary path (repeatable)")
	_ = sessionAttachCmd.MarkFlagRequired("path")

	// component flags
	componentListCmd.Flags().StringVar(&session, "session", "", "Session name or ID")
	componentGetCmd.Flags().StringVar(&session, "session", "", "Session name or ID")
	componentDescribeCmd.Flags().StringVar(&session, "session", "", "Session name or ID")

	// tree
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(componentCmd)
	rootCmd.AddCommand(healthCmd)

	sessionCmd.AddCommand(sessionCreateCmd)
	sessionCmd.AddCommand(sessionAttachCmd)
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)

	componentCmd.AddCommand(componentListCmd)
	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentDescribeCmd)

	healthCmd.AddCommand(healthCheckCmd)
}
