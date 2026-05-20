package classification

import (
	"strings"

	"github.com/sx-tane/context-os/shared/types"
)

func Classify(doc types.NormalizedDocument) types.ClassifiedDocument {
	body := strings.ToLower(doc.Title + " " + doc.Body)
	classification := types.Unknown
	score := 0.4
	switch {
	case strings.Contains(body, "blocker") || strings.Contains(body, "blocked"):
		classification, score = types.Blocker, 0.9
	case strings.Contains(body, "decision") || strings.Contains(body, "decided"):
		classification, score = types.Decision, 0.85
	case strings.Contains(body, "risk") || strings.Contains(body, "delay"):
		classification, score = types.PMORisk, 0.8
	case strings.Contains(body, "frontend") || strings.Contains(body, "fe") || strings.Contains(body, "screen"):
		classification, score = types.FEConcern, 0.75
	case strings.Contains(body, "backend") || strings.Contains(body, "be") || strings.Contains(body, "database"):
		classification, score = types.BEConcern, 0.75
	case strings.Contains(body, "api") || strings.Contains(body, "endpoint"):
		classification, score = types.APIDiscussion, 0.75
	case strings.Contains(body, "requirement") || strings.Contains(body, "business logic"):
		classification, score = types.BusinessLogic, 0.75
	}
	return types.ClassifiedDocument{Document: doc, Classification: classification, Confidence: score}
}
