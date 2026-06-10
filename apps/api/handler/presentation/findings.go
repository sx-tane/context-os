package presentation

import (
	"context"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	"context-os/domain/repository"
	"context-os/internal/pipeline"
	"context-os/internal/stages/ingestion"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
	if broad, examples := broadCodexSource(req); broad {
		response.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "source_too_broad",
			"message":  "Choose a specific repo, project, issue, channel, thread, document, or folder before running Codex-backed local analysis.",
			"examples": examples,
		})
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
		h.logAudit(r.Context(), repository.AuditEvent{
			WorkspaceID: workspaceID,
			EventType:   "ingest.started",
			Actor:       "api",
			Connector:   strings.ToLower(strings.TrimSpace(req.Connector)),
			SourceURI:   strings.TrimSpace(req.URI),
			TraceID:     stores.TraceID,
			Payload:     map[string]string{"route": "presentation.findings"},
		})
	}

	metadata := cloneMetadata(req.Metadata)
	metadata[metadataProductConnector] = strings.ToLower(strings.TrimSpace(req.Connector))
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
		if workspaceID != "" && stores != nil {
			h.logAudit(ctx, repository.AuditEvent{
				WorkspaceID: workspaceID,
				EventType:   "ingest.failed",
				Actor:       "api",
				Connector:   strings.ToLower(strings.TrimSpace(req.Connector)),
				SourceURI:   strings.TrimSpace(req.URI),
				TraceID:     stores.TraceID,
				Payload:     map[string]string{"route": "presentation.findings", "error": err.Error()},
			})
		}
		response.WriteConnectorError(w, err)
		return
	}
	actionable, reviewCandidates := splitPresentationFindings(result.Mismatches)

	// ── record connector sync cursor ───────────────────────────────────────────
	if workspaceID != "" && h.syncRepo != nil {
		syncCtx, scancel := presentationWriteContext(r.Context())
		now := time.Now().UTC()
		eventCount := result.EventCount
		if h.events != nil {
			if count, cErr := h.events.Count(syncCtx, workspaceID, strings.ToLower(strings.TrimSpace(req.Connector))); cErr == nil {
				eventCount = count
			}
		}
		_ = h.syncRepo.Upsert(syncCtx, repository.ConnectorSync{
			WorkspaceID:  workspaceID,
			Connector:    strings.ToLower(strings.TrimSpace(req.Connector)),
			SourceURI:    strings.TrimSpace(req.URI),
			LastSyncedAt: &now,
			EventCount:   eventCount,
			Status:       "idle",
		})
		scancel()
	}
	if workspaceID != "" && stores != nil {
		h.logAudit(ctx, repository.AuditEvent{
			WorkspaceID: workspaceID,
			EventType:   "ingest.completed",
			Actor:       "api",
			Connector:   strings.ToLower(strings.TrimSpace(req.Connector)),
			SourceURI:   strings.TrimSpace(req.URI),
			TraceID:     stores.TraceID,
			Payload: map[string]string{
				"route":                  "presentation.findings",
				"event_count":            strconv.Itoa(result.EventCount),
				"entity_count":           strconv.Itoa(len(result.Entities)),
				"relationship_count":     strconv.Itoa(len(result.Relationships)),
				"mismatch_count":         strconv.Itoa(len(actionable)),
				"review_candidate_count": strconv.Itoa(len(reviewCandidates)),
			},
		})
		h.logAudit(ctx, repository.AuditEvent{
			WorkspaceID: workspaceID,
			EventType:   "graph.updated",
			Actor:       "api",
			Connector:   strings.ToLower(strings.TrimSpace(req.Connector)),
			SourceURI:   strings.TrimSpace(req.URI),
			TraceID:     stores.TraceID,
			Payload: map[string]string{
				"entity_count":       strconv.Itoa(len(result.Entities)),
				"relationship_count": strconv.Itoa(len(result.Relationships)),
			},
		})
		if len(actionable) > 0 {
			h.logAudit(ctx, repository.AuditEvent{
				WorkspaceID: workspaceID,
				EventType:   "findings.detected",
				Actor:       "api",
				Connector:   strings.ToLower(strings.TrimSpace(req.Connector)),
				SourceURI:   strings.TrimSpace(req.URI),
				TraceID:     stores.TraceID,
				Payload:     map[string]string{"mismatch_count": strconv.Itoa(len(actionable))},
			})
		}
	}

	role := parseRole(req.Role)
	mismatchIDs := collectMismatchIDs(actionable)
	// Display trace incorporates mismatch content so the same findings always
	// map to the same ID (content-addressable). The run trace in stores.TraceID
	// was already used by persistResult; we only use the display trace here.
	traceID := buildTraceID(req.Connector, req.URI, mismatchIDs)
	views := buildRoleViews(actionable)

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
		executionEvidence = h.runAssistiveExecution(r.Context(), traceID, req, role, mismatchIDs, actionable)
	}

	response.WriteJSON(w, http.StatusOK, response.PresentationFindings{
		Connector:            strings.ToLower(strings.TrimSpace(req.Connector)),
		URI:                  strings.TrimSpace(req.URI),
		Role:                 string(role),
		TraceID:              traceID,
		Summary:              renderFindingsSummary(role, actionable, reviewCandidates),
		EventCount:           result.EventCount,
		EntityCount:          len(result.Entities),
		RelationshipCount:    len(result.Relationships),
		MismatchCount:        len(actionable),
		ReviewCandidateCount: len(reviewCandidates),
		SeverityCount:        severityCount(actionable),
		MismatchIDs:          mismatchIDs,
		Mismatches:           actionable,
		ReviewCandidates:     reviewCandidates,
		Views:                views,
		PMO:                  buildPMOSummary(actionable),
		Execution:            executionEvidence,
	})
}
