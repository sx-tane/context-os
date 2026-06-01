package response

import "context-os/domain/repository"

// ChatQuery is the JSON payload returned by POST /chat/query.
type ChatQuery struct {
	Intent        string                     `json:"intent"`
	WorkspaceID   string                     `json:"workspace_id"`
	WorkspacePath string                     `json:"workspace_path"`
	Connector     string                     `json:"connector,omitempty"`
	SourceURI     string                     `json:"source_uri,omitempty"`
	Answer        string                     `json:"answer"`
	Summary       string                     `json:"summary"`
	RangeStart    string                     `json:"range_start,omitempty"`
	RangeEnd      string                     `json:"range_end,omitempty"`
	ArtifactCount int                        `json:"artifact_count"`
	Artifacts     []Artifact                 `json:"artifacts"`
	Syncs         []repository.ConnectorSync `json:"syncs,omitempty"`
}
