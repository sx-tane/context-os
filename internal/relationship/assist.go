package relationship

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"context-os/domain/entities"
	"context-os/domain/types"
)

const (
	// MinAssistConfidence is the minimum confidence accepted from relationship assistance.
	MinAssistConfidence = 0.75

	// AssistOutputPrefix marks the single machine-readable line returned by Codex.
	AssistOutputPrefix = "CONTEXTOS_RELATIONSHIPS_JSON:"

	// MetadataAssistive marks accepted relationship-assist edges.
	MetadataAssistive = "assistive"
	// MetadataAssistProvider records which assist provider proposed an accepted edge.
	MetadataAssistProvider = "assist_provider"
	// MetadataAssistEvidence preserves the source quote that justified an accepted edge.
	MetadataAssistEvidence = "assist_evidence"
	// AssistProviderCodexCLI identifies the local Codex CLI relationship assistant.
	AssistProviderCodexCLI = "codex_cli"
)

// Assistant proposes additional same-document relationships between canonical entities.
type Assistant interface {
	// ProposeRelationships returns candidate relationships for doc and canonical.
	ProposeRelationships(ctx context.Context, doc types.NormalizedDocument, canonical []entities.CanonicalEntity) ([]types.Relationship, error)
}

type providerNamer interface {
	Provider() string
}

type assistEnvelope struct {
	Relationships []assistJSONRelationship `json:"relationships"`
}

type assistJSONRelationship struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	Kind       string  `json:"kind"`
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
}

// BuildWithAssist returns deterministic relationships plus accepted assistant proposals.
//
// The deterministic Build output is always preserved. Assistant failures or rejected proposals
// degrade to that baseline without deleting or mutating deterministic edges.
func BuildWithAssist(ctx context.Context, doc types.NormalizedDocument, canonical []entities.CanonicalEntity, assistant Assistant) []types.Relationship {
	base := Build(canonical)
	if assistant == nil || len(canonical) < 2 {
		return base
	}

	proposed, err := assistant.ProposeRelationships(ctx, doc, canonical)
	if err != nil {
		return base
	}

	seen := relationshipIDs(base)
	accepted := acceptAssistRelationships(assistProvider(assistant), doc, canonical, proposed, seen)
	if len(accepted) == 0 {
		return base
	}
	return append(base, accepted...)
}

// ParseAssistantOutput parses the Codex relationship JSON line and returns accepted relationships.
//
// Invalid individual proposals are rejected while valid proposals from the same line are kept.
// Malformed JSON returns an error so callers can distinguish a bad assistant response from an
// empty or fully rejected response.
func ParseAssistantOutput(output string, doc types.NormalizedDocument, canonical []entities.CanonicalEntity) ([]types.Relationship, error) {
	line, ok := assistJSONLine(output)
	if !ok {
		return nil, nil
	}

	var envelope assistEnvelope
	if err := json.Unmarshal([]byte(line), &envelope); err != nil {
		return nil, fmt.Errorf("parse relationship assist output: %w", err)
	}

	lookup := entityLookup(canonical)
	proposed := make([]types.Relationship, 0, len(envelope.Relationships))
	for _, item := range envelope.Relationships {
		from, ok := lookup.find(item.From)
		if !ok {
			continue
		}
		to, ok := lookup.find(item.To)
		if !ok {
			continue
		}
		proposed = append(proposed, types.Relationship{
			FromID:     from.ID,
			ToID:       to.ID,
			Kind:       types.RelationshipKind(strings.TrimSpace(item.Kind)),
			Confidence: item.Confidence,
			Evidence:   []string{strings.TrimSpace(item.Evidence)},
		})
	}
	return acceptAssistRelationships(AssistProviderCodexCLI, doc, canonical, proposed, map[string]struct{}{}), nil
}

func assistJSONLine(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, AssistOutputPrefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, AssistOutputPrefix)), true
		}
	}
	return "", false
}

func acceptAssistRelationships(provider string, doc types.NormalizedDocument, canonical []entities.CanonicalEntity, proposed []types.Relationship, seen map[string]struct{}) []types.Relationship {
	known := entitiesByID(canonical)
	accepted := make([]types.Relationship, 0, len(proposed))
	for _, rel := range proposed {
		from, fromOK := known[rel.FromID]
		to, toOK := known[rel.ToID]
		if !fromOK || !toOK {
			continue
		}
		if from.SourceID != doc.ID || to.SourceID != doc.ID {
			continue
		}
		if !knownRelationshipKind(rel.Kind) {
			continue
		}
		if rel.Confidence < MinAssistConfidence {
			continue
		}
		evidence, ok := acceptedEvidence(doc, rel.Evidence)
		if !ok {
			continue
		}

		rel.ID = relationshipID(rel.FromID, rel.ToID, rel.Kind)
		if err := Validate(rel); err != nil {
			continue
		}
		if _, dup := seen[rel.ID]; dup {
			continue
		}
		seen[rel.ID] = struct{}{}

		rel.Metadata = assistMetadata(rel.Metadata, doc.ID, provider, evidence)
		rel.Evidence = evidence
		accepted = append(accepted, rel)
	}
	return accepted
}

func relationshipIDs(rels []types.Relationship) map[string]struct{} {
	seen := make(map[string]struct{}, len(rels))
	for _, rel := range rels {
		seen[rel.ID] = struct{}{}
	}
	return seen
}

func assistProvider(assistant Assistant) string {
	if named, ok := assistant.(providerNamer); ok {
		if provider := strings.TrimSpace(named.Provider()); provider != "" {
			return provider
		}
	}
	return "assistant"
}

func assistMetadata(input map[string]string, sourceID, provider string, evidence []string) map[string]string {
	out := make(map[string]string, len(input)+4)
	for key, value := range input {
		out[key] = value
	}
	out["source_id"] = sourceID
	out[MetadataAssistive] = "true"
	out[MetadataAssistProvider] = provider
	out[MetadataAssistEvidence] = strings.Join(evidence, " | ")
	return out
}

func acceptedEvidence(doc types.NormalizedDocument, evidence []string) ([]string, bool) {
	if len(evidence) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(evidence))
	for _, value := range evidence {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || !evidenceInDocument(doc, trimmed) {
			return nil, false
		}
		out = append(out, trimmed)
	}
	return out, true
}

func evidenceInDocument(doc types.NormalizedDocument, evidence string) bool {
	return containsEvidence(doc.Title, evidence) || containsEvidence(doc.Body, evidence)
}

func containsEvidence(text, evidence string) bool {
	if strings.Contains(text, evidence) {
		return true
	}
	return strings.Contains(strings.Join(strings.Fields(text), " "), strings.Join(strings.Fields(evidence), " "))
}

func knownRelationshipKind(kind types.RelationshipKind) bool {
	switch kind {
	case types.CoOccursInDocument,
		types.RequirementAffectsAPI,
		types.RequirementAffectsService,
		types.APIBackedByDB,
		types.EnumConstrainsField,
		types.ServiceDependsOn:
		return true
	default:
		return false
	}
}

type canonicalLookup map[string][]types.Entity

func entityLookup(canonical []entities.CanonicalEntity) canonicalLookup {
	lookup := canonicalLookup{}
	for _, canonicalEntity := range canonical {
		entity := canonicalEntity.Entity
		lookup.add(entity.Name, entity)
		lookup.add(entity.ID, entity)
		for _, alias := range entity.Aliases {
			lookup.add(alias, entity)
		}
	}
	return lookup
}

func (l canonicalLookup) add(name string, entity types.Entity) {
	key := lookupKey(name)
	if key == "" {
		return
	}
	for _, existing := range l[key] {
		if existing.ID == entity.ID {
			return
		}
	}
	l[key] = append(l[key], entity)
}

func (l canonicalLookup) find(name string) (types.Entity, bool) {
	matches := l[lookupKey(name)]
	if len(matches) != 1 {
		return types.Entity{}, false
	}
	return matches[0], true
}

func lookupKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func entitiesByID(canonical []entities.CanonicalEntity) map[string]types.Entity {
	out := make(map[string]types.Entity, len(canonical))
	for _, canonicalEntity := range canonical {
		out[canonicalEntity.Entity.ID] = canonicalEntity.Entity
	}
	return out
}
