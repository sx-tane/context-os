package request

// WorkspaceUpsert is the JSON body accepted by POST /workspace/upsert and /workspace/reset.
type WorkspaceUpsert struct {
	// Path is the absolute local folder path for the workspace.
	Path string `json:"path"`
	// Name is the optional human-readable workspace name.
	Name string `json:"name"`
}

// WorkspaceSource is the JSON body accepted by POST /workspace/source.
type WorkspaceSource struct {
	// WorkspaceID is a workspace path or ID.
	WorkspaceID string `json:"workspace_id"`
	// Connector is the connector name, e.g. github or jira.
	Connector string `json:"connector"`
	// SourceURI is the external source URI to save.
	SourceURI string `json:"source_uri"`
	// URI is accepted for frontend compatibility with existing source forms.
	URI string `json:"uri,omitempty"`
}

// AnalysisBasket is the JSON body accepted by PUT /workspace/analysis-basket.
type AnalysisBasket struct {
	// WorkspaceID is a workspace path or ID.
	WorkspaceID string `json:"workspace_id"`
	// Items are selected evidence sources for analysis.
	Items []AnalysisBasketItem `json:"items"`
}

// AnalysisBasketItem is one selected evidence source for analysis.
type AnalysisBasketItem struct {
	ID         string `json:"id"`
	Connector  string `json:"connector"`
	URI        string `json:"uri"`
	Label      string `json:"label"`
	Origin     string `json:"origin"`
	ArtifactID string `json:"artifactId,omitempty"`
	MessageID  string `json:"messageId,omitempty"`
	AddedAt    string `json:"addedAt"`
}

// FindingActions is the JSON body accepted by PUT /workspace/finding-actions.
type FindingActions struct {
	// WorkspaceID is a workspace path or ID.
	WorkspaceID string `json:"workspace_id"`
	// Actions are durable finding checklist entries.
	Actions []FindingActionItem `json:"actions"`
}

// FindingActionItem is one durable finding checklist entry.
type FindingActionItem struct {
	FindingID string `json:"findingId"`
	Status    string `json:"status"`
	Note      string `json:"note,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}
