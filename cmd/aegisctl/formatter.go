package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
)

// ANSI — minimal palette: bold, dim, green, red, yellow.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
)

func bold(s string) string   { return ansiBold + s + ansiReset }
func dim(s string) string    { return ansiDim + s + ansiReset }
func green(s string) string  { return ansiGreen + s + ansiReset }
func yellow(s string) string { return ansiYellow + s + ansiReset }
func red(s string) string    { return ansiRed + s + ansiReset }

func colorState(s string) string {
	switch strings.ToUpper(s) {
	case "RUNNING":
		return green(s)
	case "STOPPED", "STOPPING", "ERROR":
		return red(s)
	case "INITIALIZED":
		return yellow(s)
	default:
		return s
	}
}

// prettyPrint is the single entry point called by sendCommand.
func prettyPrint(resp map[string]any) {
	status, _ := resp["status"].(string)
	cmdRaw, _ := resp["command"].(string)
	message, _ := resp["message"].(string)
	errCode, _ := resp["error_code"].(string)
	data, _ := resp["data"].(map[string]any)

	isErr := strings.ToUpper(status) != "OK"

	// ── Error path ───────────────────────────────────────────────────────────
	if isErr {
		fmt.Printf("\n%s  %s\n", red("ERR"), bold(cmdRaw))
		if errCode != "" {
			fmt.Printf("    code     %s\n", yellow(errCode))
		}
		if message != "" {
			fmt.Printf("    message  %s\n", message)
		}
		fmt.Println()
		return
	}

	// ── Command dispatch ─────────────────────────────────────────────────────
	cmd := core.CLICommandType(cmdRaw)
	switch cmd {
	case core.CommandSessionCreate:
		renderSessionCreate(data, message)
	case core.CommandSessionAttach:
		renderSessionAttach(data, message)
	case core.CommandSessionStart:
		renderSessionStart(data, message)
	case core.CommandSessionStop:
		renderSessionStop(data, message)
	case core.CommandSessionList:
		renderSessionList(data)
	case core.CommandSessionState:
		renderSessionState(data)
	case core.CommandSessionDelete:
		renderOK(cmdRaw, message)
	case core.CommandComponentList:
		renderComponentList(data, message)
	case core.CommandComponentGet, core.CommandComponentDescribe:
		renderComponentDetail(data, message)
	case core.CommandHealthCheck, core.CommandHealthCheckSession, core.CommandHealthCheckComp:
		renderHealth(data, message)
	case core.CommandDaemonShutdown, core.CommandDaemonKill:
		renderOK(cmdRaw, message)
	default:
		renderFallback(cmdRaw, message, data)
	}
}

// ── Renderers ─────────────────────────────────────────────────────────────────

func renderSessionCreate(data map[string]any, msg string) {
	sess, _ := data["session"].(map[string]any)
	fmt.Printf("\n%s  session created\n", green("OK"))
	if sess == nil {
		printMsg(msg)
		return
	}
	row("id", str(sess["id"]))
	row("name", bold(str(sess["name"])))
	row("mode", str(sess["mode"]))
	row("state", colorState(str(sess["state"])))
	row("created", fmtTime(str(sess["created_at"])))
	fmt.Println()
}

func renderSessionAttach(data map[string]any, msg string) {
	fmt.Printf("\n%s  session attach\n", green("OK"))
	row("session", str(data["session_id"]))
	if paths, ok := data["attached_components"].([]any); ok {
		row("components", fmt.Sprintf("%d attached", len(paths)))
		for _, p := range paths {
			fmt.Printf("             %s\n", dim(str(p)))
		}
	}
	fmt.Println()
}

func renderSessionStart(data map[string]any, _ string) {
	fmt.Printf("\n%s  session start\n", green("OK"))
	row("session", str(data["session_id"]))
	row("state", colorState(str(data["current_state"])))
	row("started", fmtTime(str(data["started_at"])))

	if comps, ok := data["components"].([]any); ok && len(comps) > 0 {
		fmt.Printf("\n    %-24s %-14s %s\n",
			dim("COMPONENT"), dim("STATE"), dim("VERSION"))
		fmt.Printf("    %s\n", strings.Repeat("─", 46))
		for _, c := range comps {
			printCompRow(c)
		}
	}
	fmt.Println()
}

func renderSessionStop(data map[string]any, _ string) {
	fmt.Printf("\n%s  session stop\n", green("OK"))
	row("session", str(data["session_id"]))
	row("state", colorState(str(data["current_state"])))
	if s := str(data["stopped_at"]); s != "" {
		row("stopped", fmtTime(s))
	}
	fmt.Println()
}

func renderSessionList(data map[string]any) {
	if len(data) == 0 {
		fmt.Printf("\n%s  no sessions\n\n", dim("--"))
		return
	}

	fmt.Printf("\n    %-14s %-18s %-12s %-12s %s\n",
		dim("ID"), dim("NAME"), dim("MODE"), dim("STATE"), dim("STARTED"))
	fmt.Printf("    %s\n", strings.Repeat("─", 70))

	for _, v := range data {
		sess, ok := v.(map[string]any)
		if !ok {
			continue
		}
		fmt.Printf("    %-14s %-18s %-12s %-12s %s\n",
			str(sess["id"]),
			bold(str(sess["name"])),
			str(sess["mode"]),
			colorState(str(sess["state"])),
			dim(fmtTimeShort(str(sess["started_at"]))),
		)
		if comps, ok := sess["components"].([]any); ok {
			for _, c := range comps {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				fmt.Printf("    %s %-12s %-16s %s\n",
					dim("  └"),
					dim(str(firstOf(cm, "ID", "id"))),
					dim(str(firstOf(cm, "Name", "name"))),
					colorState(str(firstOf(cm, "State", "state"))),
				)
			}
		}
	}
	fmt.Println()
}

func renderSessionState(data map[string]any) {
	fmt.Printf("\n%s  session state\n", green("OK"))
	row("session", str(data["session_id"]))
	row("state", colorState(str(data["state"])))
	fmt.Println()
}

func renderComponentList(data map[string]any, _ string) {
	list, _ := data["components"].([]any)
	if len(list) == 0 {
		fmt.Printf("\n%s  no components\n\n", dim("--"))
		return
	}
	fmt.Printf("\n    %-14s %-20s %-12s %s\n",
		dim("ID"), dim("NAME"), dim("STATE"), dim("VERSION"))
	fmt.Printf("    %s\n", strings.Repeat("─", 54))
	for _, c := range list {
		printCompRow(c)
	}
	fmt.Println()
}

func renderComponentDetail(data map[string]any, msg string) {
	fmt.Printf("\n%s  component\n", green("OK"))
	printMsg(msg)
	printKVMap(data, "    ")
	fmt.Println()
}

func renderHealth(data map[string]any, msg string) {
	fmt.Printf("\n%s  health\n", green("OK"))
	printMsg(msg)
	printKVMap(data, "    ")
	fmt.Println()
}

func renderOK(cmd, msg string) {
	fmt.Printf("\n%s  %s\n", green("OK"), strings.ToLower(strings.ReplaceAll(cmd, "_", " ")))
	printMsg(msg)
	fmt.Println()
}

func renderFallback(cmd, msg string, data map[string]any) {
	fmt.Printf("\n%s  %s\n", green("OK"), strings.ToLower(strings.ReplaceAll(cmd, "_", " ")))
	printMsg(msg)
	if len(data) > 0 {
		printKVMap(data, "    ")
	}
	fmt.Println()
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func printCompRow(raw any) {
	c, ok := raw.(map[string]any)
	if !ok {
		return
	}
	id := str(firstOf(c, "ID", "id"))
	name := str(firstOf(c, "Name", "name"))
	state := str(firstOf(c, "State", "state"))
	version := str(firstOf(c, "Version", "version"))
	fmt.Printf("    %-14s %-20s %-12s %s\n", dim(id), name, colorState(state), dim(version))
}

func row(key, val string) {
	fmt.Printf("    %-10s %s\n", dim(key), val)
}

func printMsg(msg string) {
	if msg != "" {
		fmt.Printf("    %s\n", dim(msg))
	}
}

func printKVMap(m map[string]any, indent string) {
	for k, v := range m {
		switch vv := v.(type) {
		case map[string]any:
			fmt.Printf("%s%s\n", indent, dim(k))
			printKVMap(vv, indent+"  ")
		case []any:
			fmt.Printf("%s%-18s (%d)\n", indent, dim(k), len(vv))
			for _, item := range vv {
				if mm, ok := item.(map[string]any); ok {
					printKVMap(mm, indent+"  ")
				} else {
					fmt.Printf("%s  %s\n", indent, dim(str(item)))
				}
			}
		default:
			fmt.Printf("%s%-18s %v\n", indent, dim(k), v)
		}
	}
}

// ── Time helpers ──────────────────────────────────────────────────────────────

var timeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999 -0700 MST m=+0.000000000",
	"2006-01-02 15:04:05.999999999 -0700 MST",
}

func parseTime(raw string) (time.Time, bool) {
	for _, f := range timeFormats {
		if t, err := time.Parse(f, raw); err == nil {
			return t.Local(), true
		}
	}
	return time.Time{}, false
}

func fmtTime(raw string) string {
	if raw == "" {
		return dim("—")
	}
	if t, ok := parseTime(raw); ok {
		return t.Format("2006-01-02 15:04:05")
	}
	return raw
}

func fmtTimeShort(raw string) string {
	if raw == "" {
		return "—"
	}
	if t, ok := parseTime(raw); ok {
		return t.Format("01-02 15:04:05")
	}
	return raw
}

// ── Misc ──────────────────────────────────────────────────────────────────────

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func firstOf(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}
