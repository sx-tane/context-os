package store

import "strings"

// ─── Helpers ──────────────────────────────────────────────────────────────────

// workspaceIDFromPath converts an absolute path into a stable workspace ID
// by replacing path separators with underscores and trimming leading ones.
func workspaceIDFromPath(path string) string {
	id := strings.ReplaceAll(path, "/", "_")
	id = strings.TrimPrefix(id, "_")
	if id == "" {
		return "workspace"
	}
	return id
}

// aliasesToPGArray converts a Go string slice into a PostgreSQL text array literal
// suitable for use in parameterised queries as a $N placeholder value.
func aliasesToPGArray(ss []string) string {
	if len(ss) == 0 {
		return "{}"
	}
	escaped := make([]string, len(ss))
	for i, s := range ss {
		escaped[i] = strings.ReplaceAll(s, `"`, `\"`)
	}
	return `{"` + strings.Join(escaped, `","`) + `"}`
}

// parsePGArray parses a PostgreSQL text array literal (e.g. {a,b,c}) into a
// Go string slice.  It handles the empty array and quoted elements.
func parsePGArray(raw string) []string {
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimPrefix(p, `"`)
		p = strings.TrimSuffix(p, `"`)
		p = strings.ReplaceAll(p, `\"`, `"`)
		out = append(out, p)
	}
	return out
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
