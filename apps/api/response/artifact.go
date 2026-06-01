package response

import (
	"strings"
	"time"

	"context-os/domain/repository"
)

// Artifact is one local source artifact returned by artifact and chat queries.
type Artifact struct {
	ID            string            `json:"id"`
	WorkspaceID   string            `json:"workspace_id"`
	Connector     string            `json:"connector"`
	SourceURI     string            `json:"source_uri"`
	EventType     string            `json:"event_type"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	Preview       string            `json:"preview"`
	ContentHash   string            `json:"content_hash"`
	Metadata      map[string]string `json:"metadata"`
	SchemaVersion string            `json:"schema_version"`
	IngestedAt    time.Time         `json:"ingested_at"`
}

// ArtifactList is the JSON payload returned by GET /artifacts.
type ArtifactList struct {
	WorkspaceID   string     `json:"workspace_id"`
	WorkspacePath string     `json:"workspace_path"`
	Connector     string     `json:"connector,omitempty"`
	SourceURI     string     `json:"source_uri,omitempty"`
	Query         string     `json:"query,omitempty"`
	Count         int        `json:"count"`
	Artifacts     []Artifact `json:"artifacts"`
}

// NewArtifact maps a stored ingest event into an API artifact response.
func NewArtifact(event repository.IngestEvent) Artifact {
	return Artifact{
		ID:            event.ID,
		WorkspaceID:   event.WorkspaceID,
		Connector:     event.Connector,
		SourceURI:     event.SourceURI,
		EventType:     event.EventType,
		Title:         event.Title,
		Body:          event.Body,
		Preview:       artifactPreview(event.Body),
		ContentHash:   event.ContentHash,
		Metadata:      event.Metadata,
		SchemaVersion: event.SchemaVersion,
		IngestedAt:    event.IngestedAt,
	}
}

// NewArtifacts maps stored ingest events into API artifact responses.
func NewArtifacts(events []repository.IngestEvent) []Artifact {
	artifacts := make([]Artifact, 0, len(events))
	for _, event := range events {
		artifacts = append(artifacts, NewArtifact(event))
	}
	return artifacts
}

func artifactPreview(body string) string {
	preview := strings.Join(strings.Fields(body), " ")
	if len(preview) <= 180 {
		return preview
	}
	return preview[:177] + "..."
}
