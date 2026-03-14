package main

import (
	"fmt"
	"log"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/spf13/cobra"
)

var (
	sessionMode   string
	sessionMarket string
	sessionPaths  []string
	sessionFrom   string
	sessionTo     string
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
	Short: "Start a session",
	Long: `Start a session. In historical mode, --from and --to optionally restrict the
time range of data replayed. Values are parsed as RFC3339, YYYY-MM-DD, or unix milliseconds.

Examples:
  aegisctl session start sad
  aegisctl session start sad --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z
  aegisctl session start sad --from 2024-01-01`,
	Args: cobra.ExactArgs(1),
	Run:  runSessionStart,
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop <name|id>",
	Short: "Stop a running session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionStop,
}

var sessionRestartCmd = &cobra.Command{
	Use:   "restart <name|id>",
	Short: "Restart a FINISHED session (does not relaunch component processes)",
	Long: `Restart a FINISHED session. Component processes are expected to still be running.
Optionally provide --from/--to to replay a different time range.

Examples:
  aegisctl session restart sad
  aegisctl session restart sad --from 2024-02-01T00:00:00Z`,
	Args: cobra.ExactArgs(1),
	Run:  runSessionRestart,
}

var sessionResumeCmd = &cobra.Command{
	Use:   "resume <name|id>",
	Short: "Resume a STOPPED session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionResume,
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
		payload = core.SessionCreatePayload{Name: name, Mode: sessionMode, Market: sessionMarket}
	} else {
		payload = core.SessionCreateRunPayload{Name: name, Mode: sessionMode, Market: sessionMarket, Paths: sessionPaths}
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
	from, to, err := parseTimeRange(sessionFrom, sessionTo)
	if err != nil {
		log.Fatalf("invalid time range: %s", err)
	}
	if err := sendCommand(core.CommandSessionStart, core.SessionStartPayload{
		SessionID: args[0],
		From:      from,
		To:        to,
	}); err != nil {
		log.Fatal(err)
	}
}

func runSessionStop(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandSessionStop, core.SessionActionPayload{SessionID: args[0]}); err != nil {
		log.Fatal(err)
	}
}

func runSessionRestart(_ *cobra.Command, args []string) {
	from, to, err := parseTimeRange(sessionFrom, sessionTo)
	if err != nil {
		log.Fatalf("invalid time range: %s", err)
	}
	if err := sendCommand(core.CommandSessionRestart, core.SessionStartPayload{
		SessionID: args[0],
		From:      from,
		To:        to,
	}); err != nil {
		log.Fatal(err)
	}
}

func runSessionResume(_ *cobra.Command, args []string) {
	if err := sendCommand(core.CommandSessionResume, core.SessionActionPayload{SessionID: args[0]}); err != nil {
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

// ---- helpers ----------------------------------------------------------------

func parseTimeRange(from, to string) (int64, int64, error) {
	fromTS, err := parseTimestamp(from)
	if err != nil {
		return 0, 0, fmt.Errorf("--from: %w", err)
	}
	toTS, err := parseTimestamp(to)
	if err != nil {
		return 0, 0, fmt.Errorf("--to: %w", err)
	}
	if fromTS != 0 && toTS != 0 && fromTS > toTS {
		return 0, 0, fmt.Errorf("--from must be before --to")
	}
	return fromTS, toTS, nil
}

func parseTimestamp(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UnixMilli(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC().UnixMilli(), nil
	}
	var ms int64
	if _, err := fmt.Sscanf(s, "%d", &ms); err == nil && ms > 0 {
		return ms, nil
	}
	return 0, fmt.Errorf("cannot parse %q — expected RFC3339, YYYY-MM-DD, or unix milliseconds", s)
}

// ---- flag registration ------------------------------------------------------

func init() {
	sessionCreateCmd.Flags().StringVar(&sessionMode, "mode", "historical", "Session mode (realtime|historical)")
	sessionCreateCmd.Flags().StringVar(&sessionMarket, "market", "spot", "Binance market (spot|futures|coin-m)")
	sessionCreateCmd.Flags().StringArrayVar(&sessionPaths, "path", nil, "Component binary path (repeatable)")

	sessionAttachCmd.Flags().StringArrayVar(&sessionPaths, "path", nil, "Component binary path (repeatable)")
	_ = sessionAttachCmd.MarkFlagRequired("path")

	sessionStartCmd.Flags().StringVar(&sessionFrom, "from", "", "Start of time range (RFC3339, YYYY-MM-DD, or unix ms)")
	sessionStartCmd.Flags().StringVar(&sessionTo, "to", "", "End of time range (RFC3339, YYYY-MM-DD, or unix ms)")

	sessionRestartCmd.Flags().StringVar(&sessionFrom, "from", "", "Start of time range (RFC3339, YYYY-MM-DD, or unix ms)")
	sessionRestartCmd.Flags().StringVar(&sessionTo, "to", "", "End of time range (RFC3339, YYYY-MM-DD, or unix ms)")
}
