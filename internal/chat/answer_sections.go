package chat

import (
	"encoding/json"
	"strings"
)

type liveAnswerEnvelope struct {
	Answer         string              `json:"answer"`
	Summary        string              `json:"summary"`
	AnswerSections []liveAnswerSection `json:"answer_sections"`
}

type liveAnswerSection struct {
	SourceLabel string   `json:"source_label"`
	Connector   string   `json:"connector"`
	SourceURI   string   `json:"source_uri"`
	Summary     string   `json:"summary"`
	Facts       []string `json:"facts"`
	OpenItems   []string `json:"open_items"`
	CodingNotes []string `json:"coding_notes"`
	Links       []string `json:"links"`
	Timestamps  []string `json:"timestamps"`
	Confidence  float64  `json:"confidence"`
	Status      string   `json:"status"`
}

func parseLiveAnswer(raw, fallbackConnector, fallbackSourceURI string) (string, []AnswerSection) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	var envelope liveAnswerEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return raw, nil
	}
	sections := normalizeAnswerSections(envelope.AnswerSections, fallbackConnector, fallbackSourceURI)
	answer := strings.TrimSpace(envelope.Answer)
	if answer == "" {
		answer = strings.TrimSpace(envelope.Summary)
	}
	if answer == "" {
		answer = buildAnswerFromSections(sections)
	}
	if answer == "" {
		return raw, sections
	}
	return answer, sections
}

func normalizeAnswerSections(raw []liveAnswerSection, fallbackConnector, fallbackSourceURI string) []AnswerSection {
	sections := make([]AnswerSection, 0, len(raw))
	for _, section := range raw {
		normalized := AnswerSection{
			SourceLabel: cleanSectionString(section.SourceLabel),
			Connector:   normalizeConnector(firstNonEmpty(section.Connector, fallbackConnector)),
			SourceURI:   cleanSectionString(firstNonEmpty(section.SourceURI, fallbackSourceURI)),
			Summary:     cleanSectionString(section.Summary),
			Facts:       cleanStringList(section.Facts),
			OpenItems:   cleanStringList(section.OpenItems),
			CodingNotes: cleanStringList(section.CodingNotes),
			Links:       cleanStringList(section.Links),
			Timestamps:  cleanStringList(section.Timestamps),
			Confidence:  section.Confidence,
			Status:      cleanSectionString(section.Status),
		}
		if normalized.SourceLabel == "" {
			normalized.SourceLabel = firstNonEmpty(normalized.SourceURI, normalized.Connector)
		}
		if normalized.SourceLabel == "" && normalized.Summary == "" && len(normalized.Facts) == 0 {
			continue
		}
		sections = append(sections, normalized)
	}
	return sections
}

func buildAnswerFromSections(sections []AnswerSection) string {
	parts := make([]string, 0, len(sections))
	for _, section := range sections {
		label := cleanSectionString(firstNonEmpty(section.SourceLabel, section.SourceURI, section.Connector))
		summary := cleanSectionString(section.Summary)
		if label == "" && summary == "" {
			continue
		}
		if label == "" {
			parts = append(parts, summary)
			continue
		}
		if summary == "" {
			parts = append(parts, label)
			continue
		}
		parts = append(parts, label+": "+summary)
	}
	return strings.Join(parts, "\n\n")
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if clean := cleanSectionString(value); clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func cleanSectionString(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
