package main

import (
	"log"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/spf13/cobra"
)

var (
	logFollow bool
	logAll    bool
)

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Inspect and manage components",
}

var componentListCmd = &cobra.Command{
	Use:   "list <session>",
	Short: "List all components in a session",
	Args:  cobra.ExactArgs(1),
	Run:   runComponentList,
}

var componentGetCmd = &cobra.Command{
	Use:   "get <session> [component]",
	Short: "Get component info (omit component name if only one exists)",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runComponentGet,
}

var componentDescribeCmd = &cobra.Command{
	Use:   "describe <session> [component]",
	Short: "Describe a component in detail (omit component name if only one exists)",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runComponentDescribe,
}

var componentLogsCmd = &cobra.Command{
	Use:   "logs <session> <component>",
	Short: "Stream logs for a component",
	Args:  cobra.ExactArgs(2),
	Run:   runComponentLogs,
}

// ---- handlers ---------------------------------------------------------------

func runComponentList(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandComponentList, core.ComponentListPayload{
		SessionID: args[0],
	}); err != nil {
		log.Fatal(err)
	}
}

func runComponentGet(_ *cobra.Command, args []string) {
	componentRef := ""
	if len(args) == 2 {
		componentRef = args[1]
	}
	if err := sendCommand(core.CommandComponentGet, core.ComponentGetPayload{
		SessionID:   args[0],
		ComponentID: componentRef,
	}); err != nil {
		log.Fatal(err)
	}
}

func runComponentDescribe(_ *cobra.Command, args []string) {
	componentRef := ""
	if len(args) == 2 {
		componentRef = args[1]
	}
	if err := sendCommand(core.CommandComponentDescribe, core.ComponentGetPayload{
		SessionID:   args[0],
		ComponentID: componentRef,
	}); err != nil {
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
