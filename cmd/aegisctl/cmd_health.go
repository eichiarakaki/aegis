package main

import (
	"log"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check daemon and component health",
}

var healthGlobalCmd = &cobra.Command{
	Use:   "global",
	Short: "Daemon-level health (NATS, sessions, components)",
	Args:  cobra.NoArgs,
	Run:   runHealthGlobal,
}

var healthSessionCmd = &cobra.Command{
	Use:   "session <session>",
	Short: "Session health (components, data stream, data files)",
	Args:  cobra.ExactArgs(1),
	Run:   runHealthSession,
}

var healthComponentCmd = &cobra.Command{
	Use:   "component <session> [component]",
	Short: "Component health (heartbeat, connection)",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runHealthComponent,
}

func runHealthGlobal(_ *cobra.Command, _ []string) {
	if err := sendCommand(core.CommandHealthCheck, struct{}{}); err != nil {
		log.Fatal(err)
	}
}

func runHealthSession(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandHealthCheckSession, core.SessionActionPayload{
		SessionID: args[0],
	}); err != nil {
		log.Fatal(err)
	}
}

func runHealthComponent(_ *cobra.Command, args []string) {
	ref := ""
	if len(args) == 2 {
		ref = args[1]
	}
	if err := sendCommand(core.CommandHealthCheckComp, core.ComponentGetPayload{
		SessionID:   args[0],
		ComponentID: ref,
	}); err != nil {
		log.Fatal(err)
	}
}
