package main

import (
	"log"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Health checks for Aegis subsystems",
}

var healthCheckCmd = &cobra.Command{
	Use:   "check <target>",
	Short: "Run a health check (all|data|sessions)",
	Args:  cobra.ExactArgs(1),
	Run:   runHealthCheck,
}

var healthSessionCmd = &cobra.Command{
	Use:   "session <id>",
	Short: "Health check for a specific session",
	Args:  cobra.ExactArgs(1),
	Run:   runHealthSession,
}

var healthComponentCmd = &cobra.Command{
	Use:   "component <session_id> <component_id>",
	Short: "Health check for a specific component",
	Args:  cobra.ExactArgs(2),
	Run:   runHealthComponent,
}

// ---- handlers ---------------------------------------------------------------

func runHealthCheck(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandHealthCheck, core.HealthCheckPayload{Target: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runHealthSession(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandHealthCheckSession, core.HealthCheckSessionPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runHealthComponent(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandHealthCheckComp, core.HealthCheckComponentPayload{
		SessionID:   args[0],
		ComponentID: args[1],
	}); err != nil {
		log.Fatal(err)
	}
}
