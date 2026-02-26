package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/spf13/cobra"
)

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

var (
	mode    string
	session string
)

var rootCmd = &cobra.Command{
	Use:   "aegis",
	Short: "Aegis CLI - Control plane for Aegis daemon",
}

/////////////////////////
// SESSION COMMAND
/////////////////////////

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
}

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a session",
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_START", mode)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Session started:", mode)
	},
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a session",
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_STOP", mode)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Session stopped:", mode)
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sessions",
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("SESSION_LIST", "")
		if err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// COMPONENT COMMAND
/////////////////////////

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Manage components",
}

var componentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List components",
	Run: func(cmd *cobra.Command, args []string) {
		err := sendCommand("COMPONENT_LIST", session)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var componentGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get raw component info",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := fmt.Sprintf("%s|%s", session, args[0])
		err := sendCommand("COMPONENT_GET", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var componentDescribeCmd = &cobra.Command{
	Use:   "describe [id]",
	Short: "Describe component (formatted)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := fmt.Sprintf("%s|%s", session, args[0])
		err := sendCommand("COMPONENT_DESCRIBE", payload)
		if err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// SEND FUNCTION
/////////////////////////

func sendCommand(cmdType string, payload string) error {
	cfg, err := config.LoadGlobals()
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", cfg.AegisCLISocket)
	if err != nil {
		return err
	}
	defer conn.Close()

	command := Command{
		Type:    cmdType,
		Payload: payload,
	}

	err = json.NewEncoder(conn).Encode(command)
	if err != nil {
		return err
	}

	// Read response
	var response map[string]interface{}
	err = json.NewDecoder(conn).Decode(&response)
	if err == nil {
		pretty, _ := json.MarshalIndent(response, "", "  ")
		fmt.Println(string(pretty))
	}

	return nil
}

/////////////////////////
// HEALTH COMMAND
/////////////////////////

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Health checks for Aegis subsystems",
}

var healthCheckCmd = &cobra.Command{
	Use:   "check [target]",
	Short: "Run health check (all|data|sessions)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		err := sendCommand("HEALTH_CHECK", target)
		if err != nil {
			log.Fatal(err)
		}
	},
}

/////////////////////////
// INIT
/////////////////////////

func init() {
	// Session flags
	sessionStartCmd.Flags().StringVar(&mode, "mode", "live", "Session mode (live/backtest)")
	sessionStopCmd.Flags().StringVar(&mode, "mode", "live", "Session mode (live/backtest)")

	// Component flags
	componentListCmd.Flags().StringVar(&session, "session", "live", "Session context")
	componentGetCmd.Flags().StringVar(&session, "session", "live", "Session context")
	componentDescribeCmd.Flags().StringVar(&session, "session", "live", "Session context")

	// Tree
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(componentCmd)
	rootCmd.AddCommand(healthCmd)

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	sessionCmd.AddCommand(sessionListCmd)

	componentCmd.AddCommand(componentListCmd)
	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentDescribeCmd)

	healthCmd.AddCommand(healthCheckCmd)
}
