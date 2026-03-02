package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

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
			payload := core.SessionCreatePayload{
				Name: name,
				Mode: mode,
			}
			err := sendCommand("SESSION_CREATE", payload)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		payload := core.SessionCreateRunPayload{
			Name:  name,
			Mode:  mode,
			Paths: paths,
		}
		err := sendCommand("SESSION_CREATE_RUN", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionAttachCmd = &cobra.Command{
	Use:   "attach <name|id>",
	Short: "Attach components to an existing session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := core.SessionAttachPayload{
			SessionID: args[0],
			Paths:     paths,
		}
		err := sendCommand("SESSION_ATTACH", payload)
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
		payload := core.SessionActionPayload{
			SessionID: args[0],
		}
		err := sendCommand("SESSION_START", payload)
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
		payload := core.SessionActionPayload{
			SessionID: args[0],
		}
		err := sendCommand("SESSION_STOP", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_LIST", nil)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var sessionStateCmd = &cobra.Command{
	Use:   "state <name|id>",
	Short: "Gets state of a specific session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := core.SessionActionPayload{
			SessionID: args[0],
		}
		err := sendCommand("SESSION_STATE", payload)
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
		payload := core.SessionActionPayload{
			SessionID: args[0],
		}
		err := sendCommand("SESSION_DELETE", payload)
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
	Use:   "list <session_id>",
	Short: "List components in a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := core.ComponentListPayload{
			SessionID: args[0],
		}
		err := sendCommand("COMPONENT_LIST", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var componentGetCmd = &cobra.Command{
	Use:   "get <session_id> <component_id>",
	Short: "Get raw component info",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := core.ComponentGetPayload{
			SessionID:   args[0],
			ComponentID: args[1],
		}
		err := sendCommand("COMPONENT_GET", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var componentDescribeCmd = &cobra.Command{
	Use:   "describe <session_id>",
	Short: "Describe components in a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := core.ComponentListPayload{
			SessionID: args[0],
		}
		err := sendCommand("COMPONENT_DESCRIBE", payload)
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
		payload := core.HealthCheckPayload{
			Target: args[0],
		}
		err := sendCommand("HEALTH_CHECK", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var healthSessionCmd = &cobra.Command{
	Use:   "session <id>",
	Short: "Health check for a specific session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := core.HealthCheckSessionPayload{
			SessionID: args[0],
		}
		err := sendCommand("HEALTH_CHECK_SESSION", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var healthComponentCmd = &cobra.Command{
	Use:   "component <session_id> <component_id>",
	Short: "Health check for a specific component",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := core.HealthCheckComponentPayload{
			SessionID:   args[0],
			ComponentID: args[1],
		}
		err := sendCommand("HEALTH_CHECK_COMPONENT", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// SEND
/////////////////////////

func sendCommand(cmdType string, payload interface{}) error {
	cfg, err := config.LoadGlobals()
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", cfg.AegisCLISocket)
	if err != nil {
		return err
	}
	defer conn.Close()

	requestID := uuid.NewString()

	cmd := core.Command{
		RequestID: requestID,
		Type:      cmdType,
		Payload:   payload,
	}

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
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
	sessionCreateCmd.Flags().StringVar(&mode, "mode", "historical", "Session mode (realtime/historical)")
	sessionCreateCmd.Flags().StringArrayVar(&paths, "path", []string{}, "Component binary path (repeatable)")

	// session attach flags
	sessionAttachCmd.Flags().StringArrayVar(&paths, "path", []string{}, "Component binary path (repeatable)")
	_ = sessionAttachCmd.MarkFlagRequired("path")

	// tree
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(componentCmd)
	rootCmd.AddCommand(healthCmd)

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

	healthCmd.AddCommand(healthCheckCmd)
	healthCmd.AddCommand(healthSessionCmd)
	healthCmd.AddCommand(healthComponentCmd)
}
