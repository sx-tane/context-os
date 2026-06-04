package identity

import "strings" // tokenization and convention formatting

// tokenize splits a name into lowercase word tokens, breaking on separators and
// camelCase boundaries. It is fully deterministic so convention matching never
// depends on locale or ordering. For example "refundStatus" and "refund_status"
// both tokenize to ["refund", "status"].
func tokenize(name string) []string {
	var tokens []string
	var current strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.' || r == '/': // explicit separators end a token
			tokens = appendToken(tokens, &current)
		case isUpper(r) && i > 0 && current.Len() > 0 && !isUpper(runes[i-1]): // camelCase boundary
			tokens = appendToken(tokens, &current)
			current.WriteRune(toLower(r))
		default:
			current.WriteRune(toLower(r))
		}
	}
	return appendToken(tokens, &current)
}

// appendToken flushes the builder into tokens when it holds a non-empty word.
func appendToken(tokens []string, current *strings.Builder) []string {
	if current.Len() == 0 {
		return tokens
	}
	tokens = append(tokens, current.String())
	current.Reset()
	return tokens
}

// ConventionAliases returns the canonical naming-convention variants of a name —
// snake_case, kebab-case, camelCase, PascalCase, and SCREAMING_SNAKE_CASE — so
// callers can see every spelling an entity may appear under across sources. The
// output is deterministic and omits empty results for names with no word tokens.
func ConventionAliases(name string) []string {
	tokens := tokenize(name)
	if len(tokens) == 0 {
		return nil
	}
	forms := []string{
		strings.Join(tokens, "_"),                  // snake_case
		strings.Join(tokens, "-"),                  // kebab-case
		camelCase(tokens),                          // camelCase
		pascalCase(tokens),                         // PascalCase
		strings.ToUpper(strings.Join(tokens, "_")), // SCREAMING_SNAKE_CASE
	}
	out := make([]string, 0, len(forms))
	for _, form := range forms {
		out = appendUnique(out, form)
	}
	return out
}

// camelCase joins tokens with the first lowercase and subsequent tokens title-cased.
func camelCase(tokens []string) string {
	var b strings.Builder
	for i, token := range tokens {
		if i == 0 {
			b.WriteString(token)
			continue
		}
		b.WriteString(titleToken(token))
	}
	return b.String()
}

// pascalCase joins tokens with every token title-cased.
func pascalCase(tokens []string) string {
	var b strings.Builder
	for _, token := range tokens {
		b.WriteString(titleToken(token))
	}
	return b.String()
}

// titleToken upper-cases the first rune of a token and leaves the rest unchanged.
func titleToken(token string) string {
	if token == "" {
		return token
	}
	runes := []rune(token)
	runes[0] = toUpper(runes[0])
	return string(runes)
}

// isUpper reports whether r is an ASCII uppercase letter.
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }

// toLower maps an ASCII uppercase letter to lowercase and leaves others unchanged.
func toLower(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}

// toUpper maps an ASCII lowercase letter to uppercase and leaves others unchanged.
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}
