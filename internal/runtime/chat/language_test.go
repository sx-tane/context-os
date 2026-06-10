package chat

// White-box tests cover response language normalization helpers split out of chat.go.

import "testing"

// TestResponseLanguageForMessageCorrectsEnglishHeavyMixedPrompt verifies short CJK domain terms do not force Chinese answers.
func TestResponseLanguageForMessageCorrectsEnglishHeavyMixedPrompt(t *testing.T) {
	t.Parallel()

	got := responseLanguageForMessage("zh", "status for kkg payment 決済GW linkedFlag")
	if got != "en" {
		t.Fatalf("responseLanguageForMessage() = %q, want en", got)
	}
}

// TestResponseLanguageForMessageKeepsChinesePrompt verifies real Chinese prompts keep the Chinese language hint.
func TestResponseLanguageForMessageKeepsChinesePrompt(t *testing.T) {
	t.Parallel()

	got := responseLanguageForMessage("zh", "请用中文回答最近有什么变化")
	if got != "zh" {
		t.Fatalf("responseLanguageForMessage() = %q, want zh", got)
	}
}

// TestLocalizedFallsBackToEnglish verifies unsupported language hints return the English string.
func TestLocalizedFallsBackToEnglish(t *testing.T) {
	t.Parallel()

	got := localized("fr", "english", "zh", "ja", "ko")
	if got != "english" {
		t.Fatalf("localized() = %q, want english", got)
	}
}
