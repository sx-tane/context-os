package response

import "context-os/domain/repository"

// AnswerSection is one structured source card returned by chat queries.
type AnswerSection struct {
	SourceLabel string   `json:"source_label"`
	Connector   string   `json:"connector,omitempty"`
	SourceURI   string   `json:"source_uri,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Facts       []string `json:"facts,omitempty"`
	OpenItems   []string `json:"open_items,omitempty"`
	CodingNotes []string `json:"coding_notes,omitempty"`
	Links       []string `json:"links,omitempty"`
	Timestamps  []string `json:"timestamps,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
	Status      string   `json:"status,omitempty"`
}

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
	AnswerSections []AnswerSection          `json:"answer_sections,omitempty"`
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
	// EvidenceGraphStatus describes whether saved live evidence updated the graph.
	EvidenceGraphStatus string `json:"evidence_graph_status,omitempty"`
	// EvidenceGraphEntityCount is the number of entities derived from newly saved live evidence.
	EvidenceGraphEntityCount int `json:"evidence_graph_entity_count,omitempty"`
	// EvidenceGraphRelationshipCount is the number of relationships derived from newly saved live evidence.
	EvidenceGraphRelationshipCount int `json:"evidence_graph_relationship_count,omitempty"`
}
