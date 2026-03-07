package main

import "github.com/spf13/cobra"

// rootCmd is the top-level CLI entry point.
var rootCmd = &cobra.Command{
	Use:   "aegisctl",
	Short: "Aegis - control plane for the Aegis daemon",
}

func init() {
	// Daemon sub-tree
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonShutdownCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonKillCmd)

	// Session sub-tree
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionCreateCmd)
	sessionCmd.AddCommand(sessionAttachCmd)
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	sessionCmd.AddCommand(sessionStateCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)
	sessionCmd.AddCommand(sessionRestartCmd)
	sessionCmd.AddCommand(sessionResumeCmd)

	// Component sub-tree
	rootCmd.AddCommand(componentCmd)
	componentCmd.AddCommand(componentListCmd)
	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentDescribeCmd)
	componentCmd.AddCommand(componentLogsCmd)

	// Health sub-tree
	rootCmd.AddCommand(healthCmd)
	healthCmd.AddCommand(healthGlobalCmd)
	healthCmd.AddCommand(healthSessionCmd)
	healthCmd.AddCommand(healthComponentCmd)
}
