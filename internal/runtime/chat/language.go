package chat

import (
	"strings"
)

// localized selects the translated string that matches the normalized response language.
func localized(language, english, simplifiedChinese, japanese, korean string) string {
	switch responseLanguageCode(language) {
	case "zh":
		return simplifiedChinese
	case "ja":
		return japanese
	case "ko":
		return korean
	default:
		return english
	}
}

// responseLanguageCode normalizes client language hints to supported answer language codes.
func responseLanguageCode(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "zh", "zh-cn", "cn", "zh-tw", "zh-hant":
		return "zh"
	case "ja", "jp":
		return "ja"
	case "ko", "kr":
		return "ko"
	default:
		return "en"
	}
}

// responseLanguageForMessage corrects misleading Chinese hints for English-heavy mixed prompts.
func responseLanguageForMessage(language, message string) string {
	code := responseLanguageCode(language)
	if code != "zh" {
		return code
	}
	if shouldPreferEnglishForMixedPrompt(message) {
		return "en"
	}
	return code
}

// shouldPreferEnglishForMixedPrompt detects English-heavy prompts that contain only short CJK terms.
func shouldPreferEnglishForMixedPrompt(message string) bool {
	if containsAnyRange(message, '\uac00', '\ud7af') {
		return false
	}
	if containsAnyRange(message, '\u3040', '\u30ff') {
		return false
	}
	cjkCount := countRunesInRange(message, '\u4e00', '\u9fff')
	if cjkCount == 0 || cjkCount > 6 {
		return false
	}
	if countEnglishWords(message) < 3 {
		return false
	}
	return countChineseCueRunes(message) == 0
}

// containsAnyRange reports whether text contains any rune in the inclusive range.
func containsAnyRange(value string, start, end rune) bool {
	for _, r := range value {
		if r >= start && r <= end {
			return true
		}
	}
	return false
}

// countRunesInRange counts runes inside an inclusive Unicode range.
func countRunesInRange(value string, start, end rune) int {
	count := 0
	for _, r := range value {
		if r >= start && r <= end {
			count++
		}
	}
	return count
}

// countEnglishWords counts English-like words in a mixed-language prompt.
func countEnglishWords(value string) int {
	count := 0
	inWord := false
	for _, r := range value {
		isWord := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (inWord && r >= '0' && r <= '9') || (inWord && (r == '_' || r == '-'))
		if isWord && !inWord {
			count++
		}
		inWord = isWord
	}
	return count
}

// countChineseCueRunes counts Chinese cue characters that indicate a Chinese prompt.
func countChineseCueRunes(value string) int {
	count := 0
	for _, r := range value {
		switch r {
		case '吗', '呢', '吧', '啊', '的', '了', '是', '有', '和', '在', '请', '问', '中', '文', '回', '答', '最', '近', '变', '化', '什', '么', '怎', '为':
			count++
		}
	}
	return count
}
