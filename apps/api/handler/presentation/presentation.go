// Package presentation provides HTTP handlers for role-based graph-backed summaries.
package presentation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	"context-os/domain/repository"
	"context-os/domain/types"
	"context-os/internal/execution"
	"context-os/internal/identity"
	"context-os/internal/ingestion"
	"context-os/internal/normalization"
	"context-os/internal/pipeline"
	stagepresentation "context-os/internal/presentation"
	codexsource "context-os/internal/source/codex"
	filesystemsource "context-os/internal/source/filesystem"
	githubsource "context-os/internal/source/github"
	googledrivesource "context-os/internal/source/googledrive"
	jirasource "context-os/internal/source/jira"
	notionsource "context-os/internal/source/notion"
	sharepointsource "context-os/internal/source/sharepoint"
	slacksource "context-os/internal/source/slack"
)

// findingsTimeout allows for slow Codex plugin ingestion which can take 60–90 s.
const findingsTimeout = 120 * time.Second

// findingsCacheTTL is how fresh persisted mismatches must be to be returned
// directly from DB without re-ingesting.
const findingsCacheTTL = 30 * time.Minute

// Handler holds optional repository dependencies for the presentation endpoints.
// All fields are optional — a nil field disables that capability.
type Handler struct {
	workspaces      repository.WorkspaceRepository
	events          repository.EventRepository
	mismatches      repository.MismatchRepository
	entities        repository.EntityRepository
	syncRepo        repository.SyncRepository
	parsedWriter    *normalization.DocumentWriter // persists NormalizedDocuments to storage/parsed/
	semanticMatcher identity.Matcher             // optional Layer-2 semantic identity pass
	executor        execution.CodexExecutor      // optional assistive execution backend
}

// HandlerOption configures optional capabilities on a Handler.
type HandlerOption func(*Handler)

// WithParsedWriter attaches a DocumentWriter that persists parsed documents to disk.
func WithParsedWriter(w *normalization.DocumentWriter) HandlerOption {
	return func(h *Handler) { h.parsedWriter = w }
}

// WithSemanticMatcher attaches an embedding-backed Matcher for Layer-2 identity resolution.
func WithSemanticMatcher(m identity.Matcher) HandlerOption {
	return func(h *Handler) { h.semanticMatcher = m }
}

// WithExecutor overrides the default LocalStubExecutor with a real or template-backed executor.
func WithExecutor(e execution.CodexExecutor) HandlerOption {
	return func(h *Handler) { h.executor = e }
}

// NewHandler returns a Handler wired to the provided repositories.
func NewHandler(
	workspaces repository.WorkspaceRepository,
	events repository.EventRepository,
	mismatches repository.MismatchRepository,
	entities repository.EntityRepository,
	syncRepo repository.SyncRepository,
	opts ...HandlerOption,
) *Handler {
	h := &Handler{
		workspaces: workspaces,
		events:     events,
		mismatches: mismatches,
		entities:   entities,
		syncRepo:   syncRepo,
		executor:   execution.LocalStubExecutor{}, // safe default
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Status handles GET /presentation/status.
//
// @Summary      Presentation output status
// @Description  Returns supported connectors and roles for graph-backed findings output.
// @Tags         presentation
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /presentation/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"supported_connectors": []string{"github", "jira", "slack", "filesystem", "google-drive", "notion", "sharepoint"},
		"supported_roles":      []string{"pmo", "presentation_layer", "service_layer", "qa", "architecture"},
		"execution": map[string]any{
			"hidden":    true,
			"assistive": true,
			"mode":      "local-stub",
		},
	})
}

// Findings handles POST /presentation/findings.
// This is the package-level adapter that creates a no-store Handler and delegates.
//
// @Summary      Build graph-backed role findings output
// @Description  Runs ingestion and pipeline reasoning, then renders role-specific summaries with evidence, confidence, impact, and assistive execution metadata.
// @Tags         presentation
// @Accept       json
// @Produce      json
// @Param        body  body      request.PresentationFindings  true  "Presentation findings request"
// @Success      200   {object}  response.PresentationFindings
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /presentation/findings [post]
func Findings(w http.ResponseWriter, r *http.Request) {
	h := &Handler{}
	h.Findings(w, r)
}

// Findings handles POST /presentation/findings on the stateful Handler.
// When workspace_id is provided and stores are available, results are persisted
// and cached findings are returned directly if they are within findingsCacheTTL.
func (h *Handler) Findings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.PresentationFindings
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if strings.TrimSpace(req.Connector) == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "connector is required")
		return
	}
	if strings.TrimSpace(req.URI) == "" && strings.TrimSpace(req.Content) == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri or content is required")
		return
	}

	// ── workspace wiring ───────────────────────────────────────────────────────
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	var stores *pipeline.Stores
	if workspaceID != "" && h.workspaces != nil {
		// Ensure the workspace record exists before persisting pipeline output.
		setupCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		ws, err := h.workspaces.Upsert(setupCtx, repository.Workspace{
			Name: workspaceID,
			Path: workspaceID,
		})
		cancel()
		if err == nil {
			workspaceID = ws.ID
		}

		// ── cache-hit check ────────────────────────────────────────────────────
		if !req.ForceRefresh && h.mismatches != nil {
			cacheCtx, ccancel := context.WithTimeout(r.Context(), 5*time.Second)
			cached, cErr := h.mismatches.ListByWorkspace(cacheCtx, workspaceID, "", 200)
			ccancel()
			if cErr == nil && len(cached) > 0 {
				// Check freshness: if all mismatches are younger than TTL return early.
				freshCtx, fcancel := context.WithTimeout(r.Context(), 5*time.Second)
				syncList, sErr := h.syncRepo.ListByWorkspace(freshCtx, workspaceID)
				fcancel()
				if sErr == nil {
					lastSync := lastSyncTime(syncList, strings.ToLower(strings.TrimSpace(req.Connector)))
					if !lastSync.IsZero() && time.Since(lastSync) < findingsCacheTTL {
						h.writeFindingsFromCache(w, req, workspaceID, cached)
						return
					}
				}
			}
		}

		stores = &pipeline.Stores{
			WorkspaceID: workspaceID,
			// TraceID is set here before Run so persistResult stamps every
			// persisted mismatch with a valid trace identifier.
			// The display-level trace (incorporating mismatch IDs) is computed
			// after Run and stored in the response; the run trace is stable per
			// connector+URI+timestamp to uniquely identify this execution.
			TraceID:         buildRunTraceID(req.Connector, req.URI),
			Events:          h.events,
			Entities:        h.entities,
			Mismatches:      h.mismatches,
			ParsedWriter:    h.parsedWriter,
			SemanticMatcher: h.semanticMatcher,
		}
	}

	metadata := cloneMetadata(req.Metadata)
	connector, err := resolveConnector(req, metadata)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), findingsTimeout)
	defer cancel()

	result, err := pipeline.Run(ctx, ingestion.NewPipeline(connector), contracts.SourceRequest{
		URI:      strings.TrimSpace(req.URI),
		Content:  req.Content,
		Cursor:   strings.TrimSpace(req.Cursor),
		Metadata: metadata,
	}, stores)
	if err != nil {
		response.WriteConnectorError(w, err)
		return
	}

	// ── record connector sync cursor ───────────────────────────────────────────
	if workspaceID != "" && h.syncRepo != nil {
		syncCtx, scancel := context.WithTimeout(r.Context(), 5*time.Second)
		now := time.Now().UTC()
		_ = h.syncRepo.Upsert(syncCtx, repository.ConnectorSync{
			WorkspaceID:  workspaceID,
			Connector:    strings.ToLower(strings.TrimSpace(req.Connector)),
			SourceURI:    strings.TrimSpace(req.URI),
			LastSyncedAt: &now,
			EventCount:   0,
			Status:       "idle",
		})
		scancel()
	}

	role := parseRole(req.Role)
	mismatchIDs := collectMismatchIDs(result.Mismatches)
	// Display trace incorporates mismatch content so the same findings always
	// map to the same ID (content-addressable). The run trace in stores.TraceID
	// was already used by persistResult; we only use the display trace here.
	traceID := buildTraceID(req.Connector, req.URI, mismatchIDs)
	views := buildRoleViews(result.Mismatches)

	executionEvidence := response.ExecutionEvidence{
		Enabled:   false,
		Assistive: true,
		Summary:   "execution disabled",
		Metadata: map[string]string{
			"trace_id": traceID,
			"mode":     "disabled",
		},
	}
	if req.IncludeExecution == nil || *req.IncludeExecution {
		executionEvidence = h.runAssistiveExecution(r.Context(), traceID, req, role, mismatchIDs, result.Mismatches)
	}

	response.WriteJSON(w, http.StatusOK, response.PresentationFindings{
		Connector:     strings.ToLower(strings.TrimSpace(req.Connector)),
		URI:           strings.TrimSpace(req.URI),
		Role:          string(role),
		TraceID:       traceID,
		Summary:       stagepresentation.RenderSummary(role, result.Mismatches),
		MismatchCount: len(result.Mismatches),
		SeverityCount: severityCount(result.Mismatches),
		MismatchIDs:   mismatchIDs,
		Mismatches:    result.Mismatches,
		Views:         views,
		PMO:           buildPMOSummary(result.Mismatches),
		Execution:     executionEvidence,
	})
}

// writeFindingsFromCache returns persisted mismatches without re-ingesting.
func (h *Handler) writeFindingsFromCache(w http.ResponseWriter, req request.PresentationFindings, workspaceID string, cached []types.Mismatch) {
	role := parseRole(req.Role)
	mismatchIDs := collectMismatchIDs(cached)
	traceID := buildTraceID(req.Connector, req.URI, mismatchIDs)
	views := buildRoleViews(cached)

	response.WriteJSON(w, http.StatusOK, response.PresentationFindings{
		Connector:     strings.ToLower(strings.TrimSpace(req.Connector)),
		URI:           strings.TrimSpace(req.URI),
		Role:          string(role),
		TraceID:       traceID,
		Summary:       stagepresentation.RenderSummary(role, cached),
		MismatchCount: len(cached),
		SeverityCount: severityCount(cached),
		MismatchIDs:   mismatchIDs,
		Mismatches:    cached,
		Views:         views,
		PMO:           buildPMOSummary(cached),
		Execution: response.ExecutionEvidence{
			Enabled:   false,
			Assistive: true,
			Summary:   "results from cache",
			Metadata: map[string]string{
				"trace_id":     traceID,
				"workspace_id": workspaceID,
				"source":       "cache",
			},
		},
	})
}

// lastSyncTime returns the most recent LastSyncedAt across all syncs for the given connector,
// or zero time if none found.
func lastSyncTime(syncs []repository.ConnectorSync, connector string) time.Time {
	var t time.Time
	for _, s := range syncs {
		if s.Connector != connector {
			continue
		}
		if s.LastSyncedAt != nil && s.LastSyncedAt.After(t) {
			t = *s.LastSyncedAt
		}
	}
	return t
}

func resolveConnector(req request.PresentationFindings, metadata map[string]string) (contracts.MCPSourceConnector, error) {
	connector := strings.ToLower(strings.TrimSpace(req.Connector))
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = "token"
	}

	token := strings.TrimSpace(req.Token)
	if provider == "codex" {
		if token != "" {
			metadata[codexsource.MetadataTokenOverride] = token
		}
		switch connector {
		case "github":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginGitHub
		case "jira":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginAtlassianRovo
		case "slack":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginSlack
		case "googledrive":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginGoogleDrive
		case "notion":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginNotion
		case "sharepoint":
			metadata[codexsource.MetadataPlugin] = codexsource.PluginSharePoint
		default:
			return nil, fmt.Errorf("connector %q does not support provider=codex", connector)
		}
		return codexsource.NewConnector(), nil
	}

	switch connector {
	case "github":
		setIfNotEmpty(metadata, "github_token", token)
		return githubsource.NewConnector(), nil
	case "jira":
		setIfNotEmpty(metadata, "jira_token", token)
		return jirasource.NewConnector(), nil
	case "slack":
		setIfNotEmpty(metadata, "slack_token", token)
		return slacksource.NewConnector(), nil
	case "filesystem":
		return filesystemsource.NewConnector(), nil
	case "googledrive":
		setIfNotEmpty(metadata, googledrivesource.MetadataAccessToken, token)
		return googledrivesource.NewConnector(), nil
	case "notion":
		setIfNotEmpty(metadata, notionsource.MetadataToken, token)
		return notionsource.NewConnector(), nil
	case "sharepoint":
		setIfNotEmpty(metadata, sharepointsource.MetadataAccessToken, token)
		return sharepointsource.NewConnector(), nil
	default:
		return nil, fmt.Errorf("unsupported connector %q", connector)
	}
}

func parseRole(value string) stagepresentation.Role {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(stagepresentation.PresentationLayer):
		return stagepresentation.PresentationLayer
	case string(stagepresentation.ServiceLayer):
		return stagepresentation.ServiceLayer
	case string(stagepresentation.QA):
		return stagepresentation.QA
	case string(stagepresentation.Architecture):
		return stagepresentation.Architecture
	default:
		return stagepresentation.PMO
	}
}

func buildRoleViews(mismatches []types.Mismatch) response.RoleViews {
	return response.RoleViews{
		PMO:               buildRoleSummary(stagepresentation.PMO, mismatches),
		PresentationLayer: buildRoleSummary(stagepresentation.PresentationLayer, mismatches),
		ServiceLayer:      buildRoleSummary(stagepresentation.ServiceLayer, mismatches),
		QA:                buildRoleSummary(stagepresentation.QA, mismatches),
		Architecture:      buildRoleSummary(stagepresentation.Architecture, mismatches),
	}
}

func buildRoleSummary(role stagepresentation.Role, mismatches []types.Mismatch) response.RoleSummaryView {
	return response.RoleSummaryView{
		Role:         string(role),
		Summary:      stagepresentation.RenderSummary(role, mismatches),
		MismatchIDs:  collectMismatchIDs(mismatches),
		NextActions:  collectRecommendations(mismatches),
		FindingCount: len(mismatches),
	}
}

func buildPMOSummary(mismatches []types.Mismatch) response.PMOSummary {
	summary := response.PMOSummary{
		Facts:                make([]string, 0, len(mismatches)),
		Risks:                []string{},
		Impacts:              []string{},
		Confidence:           map[string]float64{},
		Evidence:             map[string][]string{},
		RecommendedDecisions: []string{},
	}
	seenImpact := map[string]struct{}{}
	seenDecision := map[string]struct{}{}
	for _, mismatch := range mismatches {
		summary.Facts = append(summary.Facts, fmt.Sprintf("%s: %s", mismatch.ID, mismatch.Summary))
		if strings.EqualFold(mismatch.Severity, "high") {
			summary.Risks = append(summary.Risks, fmt.Sprintf("%s (%s)", mismatch.Summary, mismatch.ID))
		}
		if impact := strings.TrimSpace(mismatch.Impact); impact != "" {
			if _, ok := seenImpact[impact]; !ok {
				seenImpact[impact] = struct{}{}
				summary.Impacts = append(summary.Impacts, impact)
			}
		}
		summary.Confidence[mismatch.ID] = mismatch.Confidence
		summary.Evidence[mismatch.ID] = append([]string(nil), mismatch.Evidence...)
		if decision := strings.TrimSpace(mismatch.Recommended); decision != "" {
			if _, ok := seenDecision[decision]; !ok {
				seenDecision[decision] = struct{}{}
				summary.RecommendedDecisions = append(summary.RecommendedDecisions, decision)
			}
		}
	}
	return summary
}

func (h *Handler) runAssistiveExecution(ctx context.Context, traceID string, req request.PresentationFindings, role stagepresentation.Role, mismatchIDs []string, mismatches []types.Mismatch) response.ExecutionEvidence {
	execCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	exec := h.executor
	if exec == nil {
		exec = execution.LocalStubExecutor{}
	}

	prompt := "findings"
	result, err := exec.Analyze(execCtx, execution.CodexRequest{
		Prompt: prompt,
		Context: map[string]string{
			"trace_id":       traceID,
			"connector":      strings.ToLower(strings.TrimSpace(req.Connector)),
			"uri":            strings.TrimSpace(req.URI),
			"role":           string(role),
			"mismatch_count": strconv.Itoa(len(mismatches)),
			"mismatch_ids":   strings.Join(mismatchIDs, ","),
		},
	})

	metadata := map[string]string{"trace_id": traceID, "mode": "local-stub"}
	if err != nil {
		metadata["status"] = "error"
		return response.ExecutionEvidence{
			Enabled:   true,
			Assistive: true,
			Summary:   "assistive execution failed",
			Metadata:  metadata,
			Error:     err.Error(),
		}
	}
	for key, value := range result.Metadata {
		metadata[key] = value
	}
	metadata["status"] = "ok"
	return response.ExecutionEvidence{
		Enabled:   true,
		Assistive: true,
		Summary:   result.Summary,
		Metadata:  metadata,
	}
}

func collectMismatchIDs(mismatches []types.Mismatch) []string {
	ids := make([]string, 0, len(mismatches))
	for _, mismatch := range mismatches {
		ids = append(ids, mismatch.ID)
	}
	sort.Strings(ids)
	return ids
}

func collectRecommendations(mismatches []types.Mismatch) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, mismatch := range mismatches {
		recommended := strings.TrimSpace(mismatch.Recommended)
		if recommended == "" {
			continue
		}
		if _, ok := seen[recommended]; ok {
			continue
		}
		seen[recommended] = struct{}{}
		out = append(out, recommended)
	}
	return out
}

func severityCount(mismatches []types.Mismatch) map[string]int {
	out := map[string]int{"low": 0, "medium": 0, "high": 0}
	for _, mismatch := range mismatches {
		severity := strings.ToLower(strings.TrimSpace(mismatch.Severity))
		if severity == "" {
			severity = "medium"
		}
		out[severity]++
	}
	return out
}

func buildTraceID(connector, uri string, mismatchIDs []string) string {
	raw := strings.ToLower(strings.TrimSpace(connector)) + "|" + strings.TrimSpace(uri) + "|" + strings.Join(mismatchIDs, ",")
	sum := sha256.Sum256([]byte(raw))
	return "trace-" + hex.EncodeToString(sum[:8])
}

// buildRunTraceID generates a unique trace ID per execution using connector,
// URI, and a nanosecond timestamp so each pipeline run has a distinct identity
// even when the same connector+URI is used repeatedly.
func buildRunTraceID(connector, uri string) string {
	raw := fmt.Sprintf("%s|%s|%d", strings.ToLower(strings.TrimSpace(connector)), strings.TrimSpace(uri), time.Now().UnixNano())
	sum := sha256.Sum256([]byte(raw))
	return "run-" + hex.EncodeToString(sum[:8])
}

func cloneMetadata(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[key] = trimmed
	}
	return out
}

func setIfNotEmpty(metadata map[string]string, key, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	metadata[key] = trimmed
}
