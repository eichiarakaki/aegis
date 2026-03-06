package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eichiarakaki/aegis/internals/core"
)

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

func stateTag(s string) string {
	upper := strings.ToUpper(s)
	switch upper {
	case "RUNNING":
		return green(upper)
	case "STOPPED", "STOPPING", "ERROR":
		return red(upper)
	case "INITIALIZED":
		return yellow(upper)
	default:
		return dim(upper)
	}
}

// prettyPrint is the single entry point called by sendCommand.
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
	case core.CommandComponentGet:
		renderComponentGet(data)
	case core.CommandComponentDescribe:
		renderComponentDescribe(data)
	case core.CommandHealthCheck:
		renderHealthGlobal(data)
	case core.CommandHealthCheckSession:
		renderHealthSession(data)
	case core.CommandHealthCheckComp:
		renderHealthComponent(data)
	default:
		renderFallback(cmdRaw, message, data)
	}

	fmt.Println()
}

// ── Session renderers ─────────────────────────────────────────────────────────

func renderSessionCreate(data map[string]any) {
	sess, _ := data["session"].(map[string]any)
	if sess == nil {
		return
	}
	fmt.Printf("%s session %q created\n", green("[OK]"), str(sess["name"]))
	fmt.Printf("  id      : %s\n", str(sess["id"]))
	fmt.Printf("  mode    : %s\n", str(sess["mode"]))
	fmt.Printf("  state   : %s\n", stateTag(str(sess["state"])))
}

func renderSessionAttach(data map[string]any) {
	paths, _ := data["attached_components"].([]any)
	fmt.Printf("%s %d component(s) attached to session %s\n",
		green("[OK]"), len(paths), str(data["session_id"]))
	for _, p := range paths {
		switch v := p.(type) {
		case map[string]any:
			fmt.Printf("  + %s  %s\n", str(v["name"]), stateTag(str(v["state"])))
		default:
			fmt.Printf("  + %s\n", str(p))
		}
	}
}

func renderSessionStart(data map[string]any) {
	fmt.Printf("%s session %s  %s → %s\n",
		green("[OK]"),
		str(data["session_id"]),
		stateTag(str(data["previous_state"])),
		stateTag(str(data["current_state"])),
	)
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
		fmt.Printf("    %s %s\n",
			stateTag(str(firstOf(cm, "state", "State"))),
			str(firstOf(cm, "name", "Name")),
		)
	}
}

func renderSessionStop(data map[string]any) {
	fmt.Printf("%s session %s  %s → %s\n",
		green("[OK]"),
		str(data["session_id"]),
		stateTag(str(data["previous_state"])),
		stateTag(str(data["current_state"])),
	)
	if s := str(data["stopped_at"]); s != "" {
		fmt.Printf("  stopped : %s\n", fmtTime(s))
	}
	comps, _ := data["components"].([]any)
	for _, c := range comps {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		fmt.Printf("    %s %s\n",
			stateTag(str(firstOf(cm, "state", "State"))),
			str(firstOf(cm, "name", "Name")),
		)
	}
}

func renderSessionList(data map[string]any) {
	sessions, _ := data["sessions"].([]any)
	if len(sessions) == 0 {
		fmt.Println("no sessions found")
		return
	}
	fmt.Printf("%d session(s):\n", len(sessions))
	for _, v := range sessions {
		sess, ok := v.(map[string]any)
		if !ok {
			continue
		}
		fmt.Println()
		fmt.Printf("  %s %s %s  mode=%s\n",
			stateTag(str(sess["state"])),
			bold(str(sess["name"])),
			dim("("+str(sess["id"])+")"),
			str(sess["mode"]),
		)
		if s := str(sess["started_at"]); s != "" {
			fmt.Printf("  %-12s %s\n", dim("started"), fmtTime(s))
		}
		if n, ok := sess["component_count"]; ok {
			fmt.Printf("  %-12s %v\n", dim("components"), n)
		}
	}
}

func renderSessionState(data map[string]any) {
	sess, _ := data["session"].(map[string]any)
	comps, _ := data["components"].([]any)
	if sess == nil {
		return
	}
	fmt.Printf("%s session %q\n", green("[OK]"), str(sess["name"]))
	fmt.Printf("  id      : %s\n", str(sess["id"]))
	fmt.Printf("  mode    : %s\n", str(sess["mode"]))
	fmt.Printf("  state   : %s\n", stateTag(str(sess["state"])))
	if up := sess["uptime_seconds"]; up != nil {
		fmt.Printf("  uptime  : %s\n", fmtUptime(up))
	}
	if s := str(sess["started_at"]); s != "" {
		fmt.Printf("  started : %s\n", fmtTime(s))
	}
	if s := str(sess["stopped_at"]); s != "" {
		fmt.Printf("  stopped : %s\n", fmtTime(s))
	}
	if len(comps) == 0 {
		return
	}
	fmt.Printf("  components (%d):\n", len(comps))
	for _, c := range comps {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		fmt.Printf("    %s %s\n",
			stateTag(str(firstOf(cm, "state", "State"))),
			str(firstOf(cm, "name", "Name")),
		)
	}
}

func renderSessionDelete(data map[string]any, msg string) {
	name := str(data["session_name"])
	id := str(data["session_id"])
	if name != "" {
		fmt.Printf("%s session %q (%s) deleted\n", green("[OK]"), name, id)
	} else {
		fmt.Printf("%s session deleted\n", green("[OK]"))
		if msg != "" {
			fmt.Printf("  %s\n", dim(msg))
		}
	}
}

// ── Component renderers ───────────────────────────────────────────────────────

func renderComponentList(data map[string]any) {
	list, _ := data["components"].([]any)
	sid := str(data["session_id"])
	if len(list) == 0 {
		fmt.Printf("%s session %s  no components\n", green("[OK]"), sid)
		return
	}
	fmt.Printf("%s session %s  %d component(s):\n", green("[OK]"), sid, len(list))
	for _, c := range list {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		fmt.Printf("  %s %s %s\n",
			stateTag(str(cm["state"])),
			bold(str(cm["name"])),
			dim(str(cm["id"])),
		)
		if syms, ok := cm["supported_symbols"].([]any); ok && len(syms) > 0 {
			fmt.Printf("    symbols    : %s\n", joinAny(syms))
		}
		if tfs, ok := cm["supported_timeframes"].([]any); ok && len(tfs) > 0 {
			fmt.Printf("    timeframes : %s\n", joinAny(tfs))
		}
		if req, ok := cm["requires"].(map[string]any); ok && len(req) > 0 {
			fmt.Printf("    requires   : %s\n", joinBoolMap(req))
		}
	}
}

func renderComponentGet(data map[string]any) {
	c, _ := data["component"].(map[string]any)
	if c == nil {
		return
	}
	fmt.Printf("%s component %q\n", green("[OK]"), str(c["name"]))
	fmt.Printf("  id      : %s\n", str(c["id"]))
	fmt.Printf("  session : %s\n", str(data["session_id"]))
	fmt.Printf("  state   : %s\n", stateTag(str(c["state"])))
	if up := c["uptime_seconds"]; up != nil {
		fmt.Printf("  uptime  : %s\n", fmtUptime(up))
	}
	if s := str(c["started_at"]); s != "" {
		fmt.Printf("  started : %s\n", fmtTime(s))
	}
	if syms, ok := c["supported_symbols"].([]any); ok && len(syms) > 0 {
		fmt.Printf("  symbols    : %s\n", joinAny(syms))
	}
	if tfs, ok := c["supported_timeframes"].([]any); ok && len(tfs) > 0 {
		fmt.Printf("  timeframes : %s\n", joinAny(tfs))
	}
	if req, ok := c["requires"].(map[string]any); ok && len(req) > 0 {
		fmt.Printf("  requires   : %s\n", joinBoolMap(req))
	}
}

func renderComponentDescribe(data map[string]any) {
	c, _ := data["component"].(map[string]any)
	if c == nil {
		return
	}
	fmt.Printf("%s describe %q  session %s\n",
		green("[OK]"), str(c["name"]), str(data["session_id"]))
	fmt.Printf("  id      : %s\n", str(c["id"]))
	fmt.Printf("  state   : %s\n", stateTag(str(c["state"])))
	if req, ok := c["requires"].(map[string]any); ok && len(req) > 0 {
		fmt.Printf("  requires   : %s\n", joinBoolMap(req))
	}
	if topics, ok := c["topics_subscribed"].([]any); ok && len(topics) > 0 {
		fmt.Printf("  subscribed (%d):\n", len(topics))
		for _, t := range topics {
			fmt.Printf("    %s\n", dim(str(t)))
		}
	}
	if socket := str(c["socket"]); socket != "" {
		fmt.Printf("  socket  : %s\n", dim(socket))
	}
	if metrics, ok := c["metrics"].(map[string]any); ok && len(metrics) > 0 {
		fmt.Printf("  metrics :\n")
		if hb := str(metrics["last_heartbeat"]); hb != "" {
			fmt.Printf("    last_heartbeat : %s\n", fmtTime(hb))
		}
		if in := metrics["messages_in"]; in != nil {
			fmt.Printf("    messages_in    : %v\n", in)
		}
		if out := metrics["messages_out"]; out != nil {
			fmt.Printf("    messages_out   : %v\n", out)
		}
	}
}

// ── Health renderers ──────────────────────────────────────────────────────────

func renderHealthGlobal(data map[string]any) {
	fmt.Printf("%s health\n", green("[OK]"))
	if data == nil || len(data) == 0 {
		return
	}
	if sessions, ok := data["sessions"].(map[string]any); ok {
		total, _ := sessions["total"].(float64)
		running, _ := sessions["running"].(float64)
		fmt.Printf("  sessions   : %d total, %d running\n", int(total), int(running))
	}
	if components, ok := data["components"].(map[string]any); ok {
		total, _ := components["total"].(float64)
		running, _ := components["running"].(float64)
		fmt.Printf("  components : %d total, %d running\n", int(total), int(running))
	}
	if uptime := data["uptime_seconds"]; uptime != nil {
		fmt.Printf("  daemon     : up %s\n", fmtUptime(uptime))
	}
}

func renderHealthSession(data map[string]any) {
	sess, _ := data["session"].(map[string]any)
	if sess == nil {
		fmt.Printf("%s health session\n", green("[OK]"))
		return
	}
	fmt.Printf("%s session %q  %s\n",
		green("[OK]"), str(sess["name"]), stateTag(str(sess["state"])))
	if up := sess["uptime_seconds"]; up != nil {
		fmt.Printf("  uptime     : %s\n", fmtUptime(up))
	}
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
		fmt.Printf("    %s %s\n",
			stateTag(str(firstOf(cm, "state", "State"))),
			str(firstOf(cm, "name", "Name")),
		)
	}
}

func renderHealthComponent(data map[string]any) {
	c, _ := data["component"].(map[string]any)
	if c == nil {
		fmt.Printf("%s health component\n", green("[OK]"))
		return
	}
	fmt.Printf("%s %s  %s\n",
		green("[OK]"), str(c["name"]), stateTag(str(c["state"])))
	if hb := str(c["last_heartbeat"]); hb != "" {
		fmt.Printf("  last heartbeat : %s\n", fmtTime(hb))
	}
	if up := c["uptime_seconds"]; up != nil {
		fmt.Printf("  uptime         : %s\n", fmtUptime(up))
	}
}

// ── Fallback ──────────────────────────────────────────────────────────────────

func renderFallback(cmd, msg string, data map[string]any) {
	fmt.Printf("%s %s\n", green("[OK]"), strings.ToLower(strings.ReplaceAll(cmd, "_", " ")))
	if msg != "" {
		fmt.Printf("  %s\n", dim(msg))
	}
	if len(data) > 0 {
		printKVMap(data, "  ")
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printKVMap(m map[string]any, indent string) {
	for k, v := range m {
		switch vv := v.(type) {
		case map[string]any:
			fmt.Printf("%s%s:\n", indent, dim(k))
			printKVMap(vv, indent+"  ")
		case []any:
			if len(vv) == 0 {
				continue // skip empty arrays
			}
			fmt.Printf("%s%s: (%d)\n", indent, dim(k), len(vv))
			for _, item := range vv {
				if mm, ok := item.(map[string]any); ok {
					printKVMap(mm, indent+"  ")
				} else {
					fmt.Printf("%s  %s\n", indent, str(item))
				}
			}
		default:
			if v == nil {
				continue // skip null values
			}
			fmt.Printf("%s%-18s %v\n", indent, dim(k), v)
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

// joinBoolMap renders {"klines": true, "trades": false} as "klines, ~trades"
func joinBoolMap(m map[string]any) string {
	parts := make([]string, 0, len(m))
	for k, v := range m {
		if b, ok := v.(bool); ok && !b {
			parts = append(parts, dim("~"+k))
		} else {
			parts = append(parts, k)
		}
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
	if raw == "" {
		return ""
	}
	for _, f := range timeFormats {
		if t, err := time.Parse(f, raw); err == nil {
			return t.Local().Format("2006-01-02 15:04:05")
		}
	}
	return raw
}

func fmtUptime(v any) string {
	var secs float64
	switch n := v.(type) {
	case float64:
		secs = n
	case int64:
		secs = float64(n)
	case json.Number:
		secs, _ = n.Float64()
	default:
		return fmt.Sprintf("%v", v)
	}
	d := time.Duration(secs) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
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

// firstOf returns the first non-nil value found for any of the given keys.
func firstOf(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}
