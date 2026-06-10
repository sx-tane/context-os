package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"context-os/internal/codexio"
)

type progressBuffer struct {
	buf      *codexio.BoundedBuffer
	progress func(string)
	partial  string
}

// Write appends raw Codex output to the diagnostic buffer and emits complete progress lines.
func (p *progressBuffer) Write(data []byte) (int, error) {
	if p.buf != nil {
		_, _ = p.buf.Write(data)
	}
	text := p.partial + string(data)
	lines := strings.Split(text, "\n")
	p.partial = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		p.emit(line)
	}
	return len(data), nil
}

// Flush emits any trailing partial Codex output line.
func (p *progressBuffer) Flush() {
	p.emit(p.partial)
	p.partial = ""
}

// emit converts one Codex output line into a user-visible progress event when it is meaningful.
func (p *progressBuffer) emit(line string) {
	line = strings.TrimSpace(line)
	if line == "" || p.progress == nil {
		return
	}
	if isNoisyCodexTextLine(line) {
		return
	}
	if strings.HasPrefix(line, "{") {
		if summary := summarizeCodexJSONEvent(line); summary != "" {
			p.progress(summary)
		}
		return
	}
	if strings.HasPrefix(line, "›") || strings.HasPrefix(line, "•") {
		p.progress(line)
		return
	}
	p.progress("• " + line)
}

// isNoisyCodexTextLine suppresses Codex startup, environment, warning, and keyring noise from chat progress.
func isNoisyCodexTextLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "" {
		return true
	}
	if strings.HasPrefix(lower, "openai codex v") ||
		strings.HasPrefix(lower, "--------") ||
		strings.HasPrefix(lower, "workdir:") ||
		strings.HasPrefix(lower, "model:") ||
		strings.HasPrefix(lower, "provider:") ||
		strings.HasPrefix(lower, "approval:") ||
		strings.HasPrefix(lower, "sandbox:") ||
		strings.HasPrefix(lower, "reasoning effort:") ||
		strings.HasPrefix(lower, "reasoning summaries:") ||
		strings.HasPrefix(lower, "session id:") ||
		strings.HasPrefix(lower, "tokens used") ||
		strings.HasPrefix(lower, "user") {
		return true
	}
	for _, marker := range []string{
		" warn codex_core_",
		" warn codex_rmcp_client::oauth",
		" warn codex_rollout::",
		" warn codex_file_watcher:",
		" warn codex_mcp::",
		" error rmcp::transport::worker:",
		"warning: codex could not find bubblewrap",
		"ignoring interface.icon_small",
		"ignoring interface.icon_large",
		"ignoring interface.defaultprompt",
		"state db discrepancy",
		"failed to read oauth tokens from keyring",
		"no secret service provider or dbus session found",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// summarizeCodexJSONEvent converts one Codex JSONL event into a compact progress line.
func summarizeCodexJSONEvent(line string) string {
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return ""
	}
	eventType := cleanProgressText(stringField(event, "type"))
	if eventType == "" {
		return ""
	}
	if eventType == "session_meta" {
		return "• Codex session metadata received."
	}
	detail := codexEventDetail(event)
	if detail == "" {
		if isNoisyCodexLifecycleEvent(eventType) {
			return ""
		}
		return "• Codex: " + readableCodexEventType(eventType)
	}
	return "• Codex: " + readableCodexEventType(eventType) + " — " + detail
}

// codexEventDetail extracts the most readable short detail from a Codex JSON event payload.
func codexEventDetail(event map[string]any) string {
	for _, key := range []string{"message", "text", "summary", "status", "name", "tool_name", "call_id"} {
		if value := cleanProgressText(stringField(event, key)); value != "" {
			return truncateProgressText(value)
		}
	}
	payload, _ := event["payload"].(map[string]any)
	for _, key := range []string{"message", "text", "summary", "status", "name", "tool_name", "call_id"} {
		if value := cleanProgressText(stringField(payload, key)); value != "" {
			return truncateProgressText(value)
		}
	}
	if item, _ := payload["item"].(map[string]any); item != nil {
		if value := cleanProgressText(firstNonEmpty(
			stringField(item, "tool_name"),
			stringField(item, "name"),
			stringField(item, "title"),
			stringField(item, "command"),
			stringField(item, "status"),
			stringField(item, "type"),
		)); value != "" {
			return truncateProgressText(value)
		}
		if args, ok := item["arguments"].(map[string]any); ok {
			if value := cleanProgressText(firstNonEmpty(stringField(args, "query"), stringField(args, "q"), stringField(args, "url"))); value != "" {
				return truncateProgressText(value)
			}
		}
	}
	if delta, _ := payload["delta"].(map[string]any); delta != nil {
		if value := cleanProgressText(firstNonEmpty(stringField(delta, "message"), stringField(delta, "text"), stringField(delta, "summary"))); value != "" {
			return truncateProgressText(value)
		}
	}
	return ""
}

// isNoisyCodexLifecycleEvent identifies lifecycle-only Codex JSON events that do not add useful progress detail.
func isNoisyCodexLifecycleEvent(eventType string) bool {
	switch eventType {
	case "thread.started", "turn.started", "turn.completed", "item.started", "item.completed":
		return true
	default:
		return false
	}
}

// readableCodexEventType formats Codex event type names for chat progress output.
func readableCodexEventType(eventType string) string {
	switch eventType {
	case "tool_call":
		return "tool call"
	case "agent_message":
		return "message"
	case "reasoning":
		return "reasoning"
	default:
		return strings.ReplaceAll(eventType, "_", " ")
	}
}

// stringField returns a string representation for simple JSON object fields used in progress summaries.
func stringField(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64, bool:
		return fmt.Sprint(typed)
	default:
		return ""
	}
}

// cleanProgressText collapses whitespace around Codex progress details.
func cleanProgressText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

// truncateProgressText caps long Codex progress details to keep streaming output readable.
func truncateProgressText(value string) string {
	const maxProgressDetail = 180
	runes := []rune(value)
	if len(runes) <= maxProgressDetail {
		return value
	}
	return string(runes[:maxProgressDetail-3]) + "..."
}
