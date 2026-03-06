package main

import (
	"log"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/spf13/cobra"
)

// Flags shared across session subcommands.
var (
	sessionMode  string
	sessionPaths []string
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
}

var sessionCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a session, optionally launching components immediately",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionCreate,
}

var sessionAttachCmd = &cobra.Command{
	Use:   "attach <name|id>",
	Short: "Attach components to an existing session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionAttach,
}

var sessionStartCmd = &cobra.Command{
	Use:   "start <name|id>",
	Short: "Start a stopped session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionStart,
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop <name|id>",
	Short: "Stop a running session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionStop,
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Run:   runSessionList,
}

var sessionStateCmd = &cobra.Command{
	Use:   "state <name|id>",
	Short: "Get the state of a specific session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionState,
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <name|id>",
	Short: "Delete a session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionDelete,
}

// ---- handlers ---------------------------------------------------------------

func runSessionCreate(_ *cobra.Command, args []string) {
	name := args[0]

	var payload interface{}
	if len(sessionPaths) == 0 {
		payload = core.SessionCreatePayload{Name: name, Mode: sessionMode}
	} else {
		payload = core.SessionCreateRunPayload{Name: name, Mode: sessionMode, Paths: sessionPaths}
	}

	if err := sendCommand(core.CommandSessionCreate, payload); err != nil {
		log.Fatal(err)
	}
}

func runSessionAttach(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandSessionAttach, core.SessionAttachPayload{
		SessionID: args[0],
		Paths:     sessionPaths,
	}); err != nil {
		log.Fatal(err)
	}
}

func runSessionStart(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandSessionStart, core.SessionActionPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runSessionStop(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandSessionStop, core.SessionActionPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runSessionList(_ *cobra.Command, _ []string) {
	if err := sendCommand(core.CommandSessionList, nil); err != nil {
		log.Fatal(err)
	}
}

func runSessionState(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandSessionState, core.SessionActionPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runSessionDelete(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandSessionDelete, core.SessionActionPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

// ---- flag registration ------------------------------------------------------

func init() {
	sessionCreateCmd.Flags().StringVar(&sessionMode, "mode", "historical", "Session mode (realtime|historical)")
	sessionCreateCmd.Flags().StringArrayVar(&sessionPaths, "path", nil, "Component binary path (repeatable)")

	sessionAttachCmd.Flags().StringArrayVar(&sessionPaths, "path", nil, "Component binary path (repeatable)")
	_ = sessionAttachCmd.MarkFlagRequired("path")
}
