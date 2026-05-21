package classification

import (
	"strings" // used to lowercase and search the document body

	"context-os/domain/types" // NormalizedDocument input and ClassifiedDocument output
)

// Classify assigns a signal category and confidence score to a normalized document using keyword rules.
func Classify(doc types.NormalizedDocument) types.ClassifiedDocument {
	body := strings.ToLower(doc.Title + " " + doc.Body) // merge title and body into one lowercase string for matching
	classification := types.Unknown                      // default to Unknown if no keyword matches
	score := 0.4                                         // default confidence is low when no rule fires
	switch {
	case strings.Contains(body, "blocker") || strings.Contains(body, "blocked"):
		classification, score = types.Blocker, 0.9 // blockers are high confidence because the keyword is unambiguous
	case strings.Contains(body, "decision") || strings.Contains(body, "decided"):
		classification, score = types.Decision, 0.85 // decisions are also clear signals
	case strings.Contains(body, "risk") || strings.Contains(body, "delay"):
		classification, score = types.PMORisk, 0.8 // risk language indicates a PMO concern
	case strings.Contains(body, "frontend") || strings.Contains(body, "fe") || strings.Contains(body, "screen") || strings.Contains(body, "presentation layer"):
		classification, score = types.ConsumerConcern, 0.75 // presentation-layer keywords suggest a consumer-side concern
	case strings.Contains(body, "backend") || strings.Contains(body, "be") || strings.Contains(body, "database") || strings.Contains(body, "service layer"):
		classification, score = types.ProducerConcern, 0.75 // service-layer keywords suggest a producer-side concern
	case strings.Contains(body, "api") || strings.Contains(body, "endpoint"):
		classification, score = types.APIDiscussion, 0.75 // API keywords indicate a contract discussion
	case strings.Contains(body, "requirement") || strings.Contains(body, "business logic"):
		classification, score = types.BusinessLogic, 0.75 // requirement language signals a business rule
	}
	return types.ClassifiedDocument{Document: doc, Classification: classification, Confidence: score} // pack result and return
}
