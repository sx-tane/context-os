package chat

import (
	"context"
	"fmt"
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
)

// EvidenceSaveInput identifies a live source whose evidence should be persisted locally.
type EvidenceSaveInput struct {
	WorkspaceID string
	Connector   string
	SourceURI   string
	Answer      string
	Summary     string
}

// EvidenceSaveResult summarizes a completed evidence persistence attempt.
type EvidenceSaveResult struct {
	EventCount int
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
	plugin := codexPluginForConnector(input.Connector)
	if plugin == "" {
		return EvidenceSaveResult{}, fmt.Errorf("unsupported evidence connector %q", input.Connector)
	}
	uri := strings.TrimSpace(input.SourceURI)
	if uri == "" {
		return EvidenceSaveResult{}, fmt.Errorf("source_uri is required")
	}

	if progress != nil {
		progress("• Saving live answer evidence to Local DB...")
	}
	saveCtx, cancel := context.WithTimeout(ctx, shared.PersistentIngestTimeout)
	defer cancel()
	metadata := map[string]string{
		codexsource.MetadataPlugin: plugin,
		metadataChatEvidence:       "true",
	}
	rawEvents := []events.Event{liveAnswerEvent(input, metadata)}
	persisted, err := service.PersistEvidenceEvents(saveCtx, shared.SourceIngestInput{
		WorkspaceID: strings.TrimSpace(input.WorkspaceID),
		Connector:   strings.TrimSpace(input.Connector),
		URI:         uri,
		Metadata:    metadata,
	}, shared.CapabilityStrings(codexCapabilities()), rawEvents)
	if err != nil {
		return EvidenceSaveResult{}, err
	}
	if progress != nil {
		progress(fmt.Sprintf("• Saved %d live answer evidence event(s) to Local DB.", persisted.EventCount))
	}
	return EvidenceSaveResult{EventCount: persisted.EventCount}, nil
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
	if connector == "" || sourceURI == "" || answer == "" {
		return EvidenceSaveInput{}, false
	}
	if connector == sourceURI {
		return EvidenceSaveInput{}, false
	}
	if codexPluginForConnector(connector) == "" {
		return EvidenceSaveInput{}, false
	}
	return EvidenceSaveInput{
		WorkspaceID: strings.TrimSpace(result.WorkspaceID),
		Connector:   connector,
		SourceURI:   sourceURI,
		Answer:      answer,
		Summary:     strings.TrimSpace(result.Summary),
	}, true
}

func liveAnswerEvent(input EvidenceSaveInput, metadata map[string]string) events.Event {
	sourceURI := strings.TrimSpace(input.SourceURI)
	connector := strings.ToLower(strings.TrimSpace(input.Connector))
	title := fmt.Sprintf("Live chat evidence: %s", sourceURI)
	eventMetadata := shared.CloneStringMap(metadata)
	eventMetadata[contracts.MetadataSourceURI] = sourceURI
	eventMetadata[events.MetadataSourceID] = sourceURI
	eventMetadata["source_uri"] = sourceURI
	eventMetadata["provider"] = "codex"
	eventMetadata["evidence_kind"] = "live_chat_answer"
	if summary := strings.TrimSpace(input.Summary); summary != "" {
		eventMetadata["summary"] = summary
	}
	return events.New(
		events.DocumentIngested,
		connector,
		title,
		strings.TrimSpace(input.Answer),
		eventMetadata,
	)
}
