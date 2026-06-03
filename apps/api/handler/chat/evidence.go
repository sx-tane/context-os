package chat

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	"context-os/domain/events"
	codexsource "context-os/internal/source/codex"
)

const (
	evidenceStatusSkipped = "skipped"
	evidenceStatusSaving  = "saving"
	evidenceStatusSaved   = "saved"
	evidenceStatusError   = "error"

	metadataChatEvidence = "chat_live_evidence"
	maxEvidenceSources   = 12
)

var (
	urlPattern          = regexp.MustCompile(`https?://[^\s<>"']+`)
	jiraKeyPattern      = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
	slackChannelPattern = regexp.MustCompile(`(^|[\s(])#[A-Za-z0-9_.-]+\b`)
	gitHubRepoPattern   = regexp.MustCompile(`\b[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/(?:pull|issues)/\d+)?\b`)
)

// EvidenceSource is one concrete live source found in a Codex answer.
type EvidenceSource struct {
	Connector string
	SourceURI string
	Section   response.AnswerSection
}

// EvidenceSaveInput identifies a live source whose evidence should be persisted locally.
type EvidenceSaveInput struct {
	WorkspaceID string
	Connector   string
	SourceURI   string
	Answer      string
	Summary     string
	Sources     []EvidenceSource
}

// EvidenceSaveResult summarizes a completed evidence persistence attempt.
type EvidenceSaveResult struct {
	EventCount        int
	GraphUpdated      bool
	EntityCount       int
	RelationshipCount int
}

// EvidenceSaver persists live answer evidence discovered during chat.
type EvidenceSaver interface {
	Save(ctx context.Context, input EvidenceSaveInput, progress func(string)) (EvidenceSaveResult, error)
}

type persistentEvidenceSaver struct{}

func (p persistentEvidenceSaver) Save(ctx context.Context, input EvidenceSaveInput, progress func(string)) (EvidenceSaveResult, error) {
	service := shared.GetPersistentIngestService()
	if service == nil {
		return EvidenceSaveResult{}, fmt.Errorf("persistent ingest is not configured")
	}
	sources := normalizedEvidenceSources(input)
	if len(sources) == 0 {
		return EvidenceSaveResult{}, fmt.Errorf("concrete source_uri is required")
	}

	if progress != nil {
		progress("• Saving live answer evidence to Local DB...")
	}
	saveCtx, cancel := context.WithTimeout(ctx, shared.PersistentIngestTimeout)
	defer cancel()
	result := EvidenceSaveResult{}
	for _, source := range sources {
		plugin := codexPluginForConnector(source.Connector)
		if plugin == "" {
			return EvidenceSaveResult{}, fmt.Errorf("unsupported evidence connector %q", source.Connector)
		}
		metadata := map[string]string{
			codexsource.MetadataPlugin: plugin,
			metadataChatEvidence:       "true",
			"related_sources":          relatedSourcesMetadata(sources),
		}
		sourceInput := input
		sourceInput.Connector = source.Connector
		sourceInput.SourceURI = source.SourceURI
		sourceInput.Summary = sourceSectionSummary(source, input.Summary)
		sourceInput.Answer = sourceSectionBody(source, input.Answer)
		rawEvents := []events.Event{liveAnswerEvent(sourceInput, metadata)}
		persisted, err := service.PersistEvidenceEvents(saveCtx, shared.SourceIngestInput{
			WorkspaceID: strings.TrimSpace(input.WorkspaceID),
			Connector:   source.Connector,
			URI:         source.SourceURI,
			Metadata:    metadata,
		}, shared.CapabilityStrings(codexCapabilities()), rawEvents)
		if err != nil {
			return EvidenceSaveResult{}, err
		}
		result.EventCount += persisted.EventCount
		result.EntityCount += persisted.EntityCount
		result.RelationshipCount += persisted.RelationshipCount
		result.GraphUpdated = result.GraphUpdated || persisted.EntityCount > 0 || persisted.RelationshipCount > 0
	}
	if progress != nil {
		progress(fmt.Sprintf("• Saved %d live answer evidence artifact(s) to Local DB.", result.EventCount))
		progress(fmt.Sprintf("• Graph updated from %d saved live answer artifact(s).", result.EventCount))
	}
	return result, nil
}

func codexCapabilities() []contracts.Capability {
	return []contracts.Capability{
		contracts.CapabilityRepository,
		contracts.CapabilityIssues,
		contracts.CapabilityMessages,
		contracts.CapabilityDocs,
	}
}

func codexPluginForConnector(connector string) string {
	switch strings.ToLower(strings.TrimSpace(connector)) {
	case "github":
		return codexsource.PluginGitHub
	case "jira":
		return codexsource.PluginAtlassianRovo
	case "slack":
		return codexsource.PluginSlack
	case "googledrive":
		return codexsource.PluginGoogleDrive
	case "notion":
		return codexsource.PluginNotion
	case "sharepoint":
		return codexsource.PluginSharePoint
	default:
		return ""
	}
}

func evidenceSaveInput(result response.ChatQuery) (EvidenceSaveInput, bool) {
	if strings.ToLower(strings.TrimSpace(result.Provider)) != "codex" {
		return EvidenceSaveInput{}, false
	}
	connector := strings.ToLower(strings.TrimSpace(result.Connector))
	sourceURI := strings.TrimSpace(result.SourceURI)
	answer := strings.TrimSpace(result.Answer)
	if connector == "" || answer == "" {
		return EvidenceSaveInput{}, false
	}
	if codexPluginForConnector(connector) == "" {
		return EvidenceSaveInput{}, false
	}
	sources := extractEvidenceSources(result)
	if len(sources) == 0 {
		return EvidenceSaveInput{}, false
	}
	return EvidenceSaveInput{
		WorkspaceID: strings.TrimSpace(result.WorkspaceID),
		Connector:   connector,
		SourceURI:   sourceURI,
		Answer:      answer,
		Summary:     strings.TrimSpace(result.Summary),
		Sources:     sources,
	}, true
}

func liveAnswerEvent(input EvidenceSaveInput, metadata map[string]string) events.Event {
	sourceURI := strings.TrimSpace(input.SourceURI)
	connector := strings.ToLower(strings.TrimSpace(input.Connector))
	eventMetadata := shared.CloneStringMap(metadata)
	eventMetadata[contracts.MetadataSourceURI] = sourceURI
	eventMetadata[contracts.MetadataConnector] = connector
	eventMetadata[events.MetadataSourceID] = sourceURI
	eventMetadata["source_uri"] = sourceURI
	eventMetadata["provider"] = "codex"
	eventMetadata["evidence_kind"] = "live_chat_answer"
	if label := strings.TrimSpace(sourceLabelFromInput(input)); label != "" {
		eventMetadata["source_label"] = label
	}
	if summary := strings.TrimSpace(input.Summary); summary != "" {
		eventMetadata["summary"] = summary
	}
	return events.New(
		events.DocumentIngested,
		connector,
		liveAnswerTitle(input),
		strings.TrimSpace(input.Answer),
		eventMetadata,
	)
}

func extractEvidenceSources(result response.ChatQuery) []EvidenceSource {
	connector := strings.ToLower(strings.TrimSpace(result.Connector))
	sourceURI := trimEvidenceSource(result.SourceURI)
	answer := strings.TrimSpace(result.Answer)
	builder := evidenceSourceBuilder{}
	for _, section := range result.AnswerSections {
		sectionConnector := strings.ToLower(strings.TrimSpace(firstNonEmpty(section.Connector, connector)))
		sectionSourceURI := trimEvidenceSource(firstNonEmpty(section.SourceURI, sourceURI))
		builder.addSection(sectionConnector, sectionSourceURI, section)
	}
	if len(builder.items) > 0 {
		return builder.sources()
	}
	if sourceURI != "" && !isBroadConnectorScope(connector, sourceURI) {
		builder.add(connector, sourceURI)
	}
	for _, match := range urlPattern.FindAllString(answer, -1) {
		uri := trimEvidenceSource(match)
		if sourceConnector := connectorForURL(uri); sourceConnector != "" {
			builder.add(sourceConnector, uri)
		}
	}
	urlRanges := urlPattern.FindAllStringIndex(answer, -1)
	for _, loc := range jiraKeyPattern.FindAllStringIndex(answer, -1) {
		if rangeContains(urlRanges, loc[0], loc[1]) {
			continue
		}
		builder.add("jira", answer[loc[0]:loc[1]])
	}
	if strings.Contains(strings.ToLower(answer), "slack") {
		for _, match := range slackChannelPattern.FindAllStringSubmatch(answer, -1) {
			if len(match) > 0 {
				builder.add("slack", strings.TrimSpace(match[0]))
			}
		}
	}
	if strings.Contains(strings.ToLower(answer), "github") {
		for _, match := range gitHubRepoPattern.FindAllString(answer, -1) {
			if strings.Contains(match, "://") {
				continue
			}
			builder.add("github", match)
		}
	}
	return builder.sources()
}

func rangeContains(ranges [][]int, start, end int) bool {
	for _, item := range ranges {
		if len(item) != 2 {
			continue
		}
		if start >= item[0] && end <= item[1] {
			return true
		}
	}
	return false
}

type evidenceSourceBuilder struct {
	items []EvidenceSource
	seen  map[string]struct{}
}

func (b *evidenceSourceBuilder) add(connector, sourceURI string) {
	b.addSection(connector, sourceURI, response.AnswerSection{})
}

func (b *evidenceSourceBuilder) addSection(connector, sourceURI string, section response.AnswerSection) {
	if b.seen == nil {
		b.seen = map[string]struct{}{}
	}
	connector = strings.ToLower(strings.TrimSpace(connector))
	sourceURI = trimEvidenceSource(sourceURI)
	if connector == "" || sourceURI == "" || isBroadConnectorScope(connector, sourceURI) {
		return
	}
	if codexPluginForConnector(connector) == "" {
		return
	}
	key := connector + "\x00" + sourceURI
	if _, ok := b.seen[key]; ok {
		return
	}
	if len(b.items) >= maxEvidenceSources {
		return
	}
	b.seen[key] = struct{}{}
	section.Connector = connector
	section.SourceURI = sourceURI
	b.items = append(b.items, EvidenceSource{Connector: connector, SourceURI: sourceURI, Section: section})
}

func (b *evidenceSourceBuilder) sources() []EvidenceSource {
	out := make([]EvidenceSource, len(b.items))
	copy(out, b.items)
	return out
}

func normalizedEvidenceSources(input EvidenceSaveInput) []EvidenceSource {
	builder := evidenceSourceBuilder{}
	for _, source := range input.Sources {
		builder.addSection(source.Connector, source.SourceURI, source.Section)
	}
	if len(builder.items) == 0 {
		builder.add(input.Connector, input.SourceURI)
	}
	return builder.sources()
}

func sourceSectionSummary(source EvidenceSource, fallback string) string {
	if summary := strings.TrimSpace(source.Section.Summary); summary != "" {
		return summary
	}
	return strings.TrimSpace(fallback)
}

func sourceSectionBody(source EvidenceSource, fallback string) string {
	section := source.Section
	if isEmptyAnswerSection(section) {
		return strings.TrimSpace(fallback)
	}
	var parts []string
	if section.SourceLabel != "" {
		parts = append(parts, "Source: "+section.SourceLabel)
	}
	if section.Summary != "" {
		parts = append(parts, "Summary: "+section.Summary)
	}
	appendSectionList := func(label string, values []string) {
		if len(values) == 0 {
			return
		}
		parts = append(parts, label+":")
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				parts = append(parts, "- "+value)
			}
		}
	}
	appendSectionList("Facts", section.Facts)
	appendSectionList("Open items", section.OpenItems)
	appendSectionList("Coding notes", section.CodingNotes)
	appendSectionList("Links", section.Links)
	appendSectionList("Timestamps", section.Timestamps)
	if section.Status != "" {
		parts = append(parts, "Status: "+section.Status)
	}
	if section.Confidence > 0 {
		parts = append(parts, fmt.Sprintf("Confidence: %.2f", section.Confidence))
	}
	return strings.Join(parts, "\n")
}

func liveAnswerTitle(input EvidenceSaveInput) string {
	if label := strings.TrimSpace(sourceLabelFromInput(input)); label != "" {
		return fmt.Sprintf("Live chat evidence: %s", label)
	}
	return fmt.Sprintf("Live chat evidence: %s", strings.TrimSpace(input.SourceURI))
}

func sourceLabelFromInput(input EvidenceSaveInput) string {
	for _, source := range input.Sources {
		if strings.EqualFold(source.Connector, input.Connector) && strings.TrimSpace(source.SourceURI) == strings.TrimSpace(input.SourceURI) {
			return source.Section.SourceLabel
		}
	}
	return ""
}

func isEmptyAnswerSection(section response.AnswerSection) bool {
	return strings.TrimSpace(section.SourceLabel) == "" &&
		strings.TrimSpace(section.Summary) == "" &&
		len(section.Facts) == 0 &&
		len(section.OpenItems) == 0 &&
		len(section.CodingNotes) == 0 &&
		len(section.Links) == 0 &&
		len(section.Timestamps) == 0 &&
		strings.TrimSpace(section.Status) == "" &&
		section.Confidence == 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func connectorForURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Host)
	path := strings.ToLower(parsed.Path)
	switch {
	case strings.Contains(host, "docs.google.com"), strings.Contains(host, "drive.google.com"):
		return "googledrive"
	case strings.Contains(host, "atlassian.net") || strings.Contains(path, "/browse/"):
		return "jira"
	case strings.Contains(host, "slack.com"):
		return "slack"
	case strings.Contains(host, "github.com"), strings.Contains(host, "api.github.com"):
		return "github"
	case strings.Contains(host, "notion.so"), strings.Contains(host, "notion.site"):
		return "notion"
	case strings.Contains(host, "sharepoint.com"), strings.Contains(host, "onedrive.live.com"):
		return "sharepoint"
	default:
		return ""
	}
}

func trimEvidenceSource(value string) string {
	return strings.Trim(strings.TrimSpace(value), ".,;:!?\"'`)]}>")
}

func isBroadConnectorScope(connector, sourceURI string) bool {
	return strings.EqualFold(strings.TrimSpace(connector), strings.TrimSpace(sourceURI))
}

func relatedSourcesMetadata(sources []EvidenceSource) string {
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		parts = append(parts, source.Connector+":"+source.SourceURI)
	}
	return strings.Join(parts, ",")
}
