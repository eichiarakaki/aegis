package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
)

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"
	ansiDim   = "\033[2m"
	ansiGreen = "\033[32m"
	ansiRed   = "\033[31m"
)

func bold(s string) string  { return ansiBold + s + ansiReset }
func dim(s string) string   { return ansiDim + s + ansiReset }
func green(s string) string { return ansiGreen + s + ansiReset }
func red(s string) string   { return ansiRed + s + ansiReset }

func prettyPrint(resp map[string]any) {
	status, _ := resp["status"].(string)
	cmdRaw, _ := resp["command"].(string)
	message, _ := resp["message"].(string)
	errCode, _ := resp["error_code"].(string)
	data, _ := resp["data"].(map[string]any)

	fmt.Println()

	if strings.ToUpper(status) != "OK" {
		fmt.Printf("%s %s\n", red("[FAIL]"), bold(cmdRaw))
		if errCode != "" {
			fmt.Printf("  error   : %s\n", errCode)
		}
		if message != "" {
			fmt.Printf("  message : %s\n", message)
		}
		fmt.Println()
		return
	}

	cmd := core.CLICommandType(cmdRaw)
	switch cmd {
	case core.CommandSessionCreate:
		renderSessionCreate(data)
	case core.CommandSessionAttach:
		renderSessionAttach(data)
	case core.CommandSessionStart:
		renderSessionStart(data)
	case core.CommandSessionStop:
		renderSessionStop(data)
	case core.CommandSessionList:
		renderSessionList(data)
	case core.CommandSessionState:
		renderSessionState(data)
	case core.CommandSessionDelete:
		renderSessionDelete(data, message)
	case core.CommandComponentList:
		renderComponentList(data)
	case core.CommandComponentGet, core.CommandComponentDescribe:
		renderComponentDetail(data)
	case core.CommandHealthCheck, core.CommandHealthCheckSession, core.CommandHealthCheckComp:
		renderHealth(data, message)
	default:
		renderFallback(cmdRaw, message, data)
	}

	fmt.Println()
}

// ── Renderers ─────────────────────────────────────────────────────────────────

func renderSessionCreate(data map[string]any) {
	sess, _ := data["session"].(map[string]any)
	if sess == nil {
		return
	}
	fmt.Printf("%s session %q created\n", green("[OK]"), str(sess["name"]))
	fmt.Printf("  id      : %s\n", str(sess["id"]))
	fmt.Printf("  mode    : %s\n", str(sess["mode"]))
	fmt.Printf("  state   : %s\n", str(sess["state"]))
}

func renderSessionAttach(data map[string]any) {
	paths, _ := data["attached_components"].([]any)
	fmt.Printf("%s %d component(s) attached to session %s\n",
		green("[OK]"), len(paths), str(data["session_id"]))
	for _, p := range paths {
		fmt.Printf("  + %s\n", str(p))
	}
}

func renderSessionStart(data map[string]any) {
	fmt.Printf("%s session %s is now %s\n",
		green("[OK]"), str(data["session_id"]), bold(str(data["current_state"])))
	fmt.Printf("  started : %s\n", fmtTime(str(data["started_at"])))

	comps, _ := data["components"].([]any)
	if len(comps) == 0 {
		return
	}
	fmt.Printf("  components (%d):\n", len(comps))
	for _, c := range comps {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		name := str(firstOf(cm, "Name", "name"))
		id := str(firstOf(cm, "ID", "id"))
		state := str(firstOf(cm, "State", "state"))
		ver := str(firstOf(cm, "Version", "version"))
		fmt.Printf("    [%s] %s (%s) v%s\n", state, name, id, ver)

		if caps, ok := cm["Capabilities"].(map[string]any); ok {
			if streams, ok := caps["requires_streams"].([]any); ok {
				fmt.Printf("      streams    : %s\n", joinAny(streams))
			}
			if symbols, ok := caps["supported_symbols"].([]any); ok {
				fmt.Printf("      symbols    : %s\n", joinAny(symbols))
			}
			if tframes, ok := caps["supported_timeframes"].([]any); ok {
				fmt.Printf("      timeframes : %s\n", joinAny(tframes))
			}
		}
	}
}

func renderSessionStop(data map[string]any) {
	fmt.Printf("%s session %s is now %s\n",
		green("[OK]"), str(data["session_id"]), bold(str(data["current_state"])))
	if s := str(data["stopped_at"]); s != "" {
		fmt.Printf("  stopped : %s\n", fmtTime(s))
	}
}

func renderSessionList(data map[string]any) {
	if len(data) == 0 {
		fmt.Println("no sessions found")
		return
	}
	fmt.Printf("%d session(s):\n", len(data))
	for _, v := range data {
		sess, ok := v.(map[string]any)
		if !ok {
			continue
		}
		fmt.Printf("\n  [%s] %s (%s)  mode=%s\n",
			str(sess["state"]),
			bold(str(sess["name"])),
			str(sess["id"]),
			str(sess["mode"]),
		)
		if s := str(sess["started_at"]); s != "" {
			fmt.Printf("    started : %s\n", fmtTime(s))
		}
		if topics, ok := sess["topics"].([]any); ok && len(topics) > 0 {
			fmt.Printf("    topics  : %s\n", joinAny(topics))
		}
		if comps, ok := sess["components"].([]any); ok && len(comps) > 0 {
			fmt.Printf("    components (%d):\n", len(comps))
			for _, c := range comps {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				fmt.Printf("      [%s] %s (%s)\n",
					str(firstOf(cm, "State", "state")),
					str(firstOf(cm, "Name", "name")),
					str(firstOf(cm, "ID", "id")),
				)
			}
		}
	}
}

func renderSessionState(data map[string]any) {
	fmt.Printf("%s session %s  state=%s\n",
		green("[OK]"), str(data["session_id"]), bold(str(data["state"])))
}

func renderSessionDelete(data map[string]any, msg string) {
	fmt.Printf("%s session deleted\n", green("[OK]"))
	if msg != "" {
		fmt.Printf("  %s\n", msg)
	}
}

func renderComponentList(data map[string]any) {
	list, _ := data["components"].([]any)
	if len(list) == 0 {
		fmt.Println("no components found")
		return
	}
	fmt.Printf("%d component(s):\n", len(list))
	for _, c := range list {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		fmt.Printf("  [%s] %s (%s) v%s\n",
			str(firstOf(cm, "State", "state")),
			str(firstOf(cm, "Name", "name")),
			str(firstOf(cm, "ID", "id")),
			str(firstOf(cm, "Version", "version")),
		)
	}
}

func renderComponentDetail(data map[string]any) {
	printKVMap(data, "  ")
}

func renderHealth(data map[string]any, msg string) {
	fmt.Printf("%s health check\n", green("[OK]"))
	if msg != "" {
		fmt.Printf("  %s\n", msg)
	}
	printKVMap(data, "  ")
}

func renderFallback(cmd, msg string, data map[string]any) {
	fmt.Printf("%s %s\n", green("[OK]"), cmd)
	if msg != "" {
		fmt.Printf("  %s\n", msg)
	}
	printKVMap(data, "  ")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printKVMap(m map[string]any, indent string) {
	for k, v := range m {
		switch vv := v.(type) {
		case map[string]any:
			fmt.Printf("%s%s:\n", indent, k)
			printKVMap(vv, indent+"  ")
		case []any:
			fmt.Printf("%s%s: %s\n", indent, k, joinAny(vv))
		default:
			fmt.Printf("%s%s: %v\n", indent, k, v)
		}
	}
}

func joinAny(items []any) string {
	parts := make([]string, 0, len(items))
	for _, i := range items {
		parts = append(parts, str(i))
	}
	return strings.Join(parts, ", ")
}

var timeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999 -0700 MST m=+0.000000000",
	"2006-01-02 15:04:05.999999999 -0700 MST",
}

func fmtTime(raw string) string {
	for _, f := range timeFormats {
		if t, err := time.Parse(f, raw); err == nil {
			return t.Local().Format("2006-01-02 15:04:05")
		}
	}
	return raw
}

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
