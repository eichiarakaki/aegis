package main

import (
	"log"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/spf13/cobra"
)

// Flags for the logs subcommand.
var (
	logFollow bool
	logAll    bool
)

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Inspect and manage components",
}

var componentListCmd = &cobra.Command{
	Use:   "list <session_id>",
	Short: "List components in a session",
	Args:  cobra.ExactArgs(1),
	Run:   runComponentList,
}

var componentGetCmd = &cobra.Command{
	Use:   "get <session_id> <component_id>",
	Short: "Get raw component info",
	Args:  cobra.ExactArgs(2),
	Run:   runComponentGet,
}

var componentDescribeCmd = &cobra.Command{
	Use:   "describe <session_id>",
	Short: "Describe all components in a session",
	Args:  cobra.ExactArgs(1),
	Run:   runComponentDescribe,
}

var componentLogsCmd = &cobra.Command{
	Use:   "logs <session_id> <component_id|name>",
	Short: "Stream logs for a component (similar to docker logs)",
	Args:  cobra.ExactArgs(2),
	Run:   runComponentLogs,
}

// ---- handlers ---------------------------------------------------------------

func runComponentList(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandComponentList, core.ComponentListPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runComponentGet(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandComponentGet, core.ComponentGetPayload{
		SessionID:   args[0],
		ComponentID: args[1],
	}); err != nil {
		log.Fatal(err)
	}
}

func runComponentDescribe(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandComponentDescribe, core.ComponentListPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runComponentLogs(_ *cobra.Command, args []string) {
	if err := streamComponentLogs(args[0], args[1], logFollow, logAll); err != nil {
		log.Fatal(err)
	}
}

// ---- flag registration ------------------------------------------------------

func init() {
	componentLogsCmd.Flags().BoolVarP(&logFollow, "follow", "f", false, "Follow log output (like tail -f)")
	componentLogsCmd.Flags().BoolVarP(&logAll, "all", "a", false, "Show all logs from the beginning of the file")
}
