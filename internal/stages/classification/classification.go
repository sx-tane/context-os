package classification

import (
	"sort"    // used to order multi-label results by confidence
	"strings" // used to lowercase and search the document body

	"context-os/domain/types" // NormalizedDocument input and ClassifiedDocument output
)

// unknownConfidence is the low score assigned when no classification rule fires.
const unknownConfidence = 0.4

// rule is one ordered keyword classification rule. Earlier rules take precedence for the primary label.
type rule struct {
	name           string               // stable rule identifier surfaced as evidence provenance
	classification types.Classification // category this rule assigns when it fires
	confidence     float64              // confidence score for a match
	keywords       []string             // any keyword presence triggers the rule
}

// rules lists classification rules in descending priority order.
var rules = []rule{
	{name: "blocker_keyword", classification: types.Blocker, confidence: 0.9, keywords: []string{"blocker", "blocked"}},
	{name: "decision_keyword", classification: types.Decision, confidence: 0.85, keywords: []string{"decision", "decided"}},
	{name: "pmo_risk_keyword", classification: types.PMORisk, confidence: 0.8, keywords: []string{"risk", "delay"}},
	{name: "consumer_concern_keyword", classification: types.ConsumerConcern, confidence: 0.75, keywords: []string{"frontend", "fe", "screen", "presentation layer"}},
	{name: "producer_concern_keyword", classification: types.ProducerConcern, confidence: 0.75, keywords: []string{"backend", "be", "database", "service layer"}},
	{name: "api_discussion_keyword", classification: types.APIDiscussion, confidence: 0.75, keywords: []string{"api", "endpoint"}},
	{name: "business_logic_keyword", classification: types.BusinessLogic, confidence: 0.75, keywords: []string{"requirement", "business logic"}},
}

// Classify assigns a primary signal category plus every matching label, with confidence and evidence.
func Classify(doc types.NormalizedDocument) types.ClassifiedDocument {
	body := strings.ToLower(doc.Title + " " + doc.Body) // merge title and body into one lowercase string for matching

	labels := []types.ScoredLabel{} // every rule that fired, in priority order
	matchedRules := []string{}      // names of all rules that fired, for explainability
	for _, r := range rules {
		matched := matchedKeywords(body, r.keywords) // keywords from this rule present in the text
		if len(matched) == 0 {
			continue // rule did not fire
		}
		labels = append(labels, types.ScoredLabel{
			Classification: r.classification,
			Confidence:     r.confidence,
			Rule:           r.name,
			Evidence:       matched,
		})
		matchedRules = append(matchedRules, r.name)
	}

	if len(labels) == 0 {
		return types.ClassifiedDocument{
			Document:       doc,
			Classification: types.Unknown,
			Confidence:     unknownConfidence,
			Labels:         []types.ScoredLabel{},
			MatchedRules:   []string{},
			Evidence:       []string{},
		}
	}

	primary := labels[0] // first matching rule wins the primary label because rules are priority-ordered

	ordered := make([]types.ScoredLabel, len(labels))
	copy(ordered, labels)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Confidence > ordered[j].Confidence // present strongest signals first while keeping rule order on ties
	})

	return types.ClassifiedDocument{
		Document:       doc,
		Classification: primary.Classification,
		Confidence:     primary.Confidence,
		Labels:         ordered,
		MatchedRules:   matchedRules,
		Evidence:       primary.Evidence,
	}
}

// matchedKeywords returns the keywords from the rule that appear in the lowercased text.
func matchedKeywords(body string, keywords []string) []string {
	matched := []string{}
	for _, kw := range keywords {
		if strings.Contains(body, kw) {
			matched = append(matched, kw)
		}
	}
	return matched
}
