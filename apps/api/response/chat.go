package response

import "context-os/domain/repository"

// ChatQuery is the JSON payload returned by POST /chat/query.
type ChatQuery struct {
	Intent        string                     `json:"intent"`
	WorkspaceID   string                     `json:"workspace_id"`
	WorkspacePath string                     `json:"workspace_path"`
	Connector     string                     `json:"connector,omitempty"`
	SourceURI     string                     `json:"source_uri,omitempty"`
	Provider      string                     `json:"provider"`
	Answer        string                     `json:"answer"`
	Summary       string                     `json:"summary"`
	RangeStart    string                     `json:"range_start,omitempty"`
	RangeEnd      string                     `json:"range_end,omitempty"`
	ArtifactCount int                        `json:"artifact_count"`
	Artifacts     []Artifact                 `json:"artifacts"`
	Syncs         []repository.ConnectorSync `json:"syncs,omitempty"`
	// EvidenceSaveStatus describes whether a live Codex answer also persisted local evidence.
	EvidenceSaveStatus string `json:"evidence_save_status,omitempty"`
	// EvidenceEventCount is the number of local events produced by the evidence save.
	EvidenceEventCount int `json:"evidence_event_count,omitempty"`
	// EvidenceSaveError is set when evidence persistence failed without discarding the live answer.
	EvidenceSaveError string `json:"evidence_save_error,omitempty"`
}
