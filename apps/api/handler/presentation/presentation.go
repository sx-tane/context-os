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
	"context-os/domain/types"
	"context-os/internal/execution"
	"context-os/internal/ingestion"
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
	}, nil)
	if err != nil {
		response.WriteConnectorError(w, err)
		return
	}

	role := parseRole(req.Role)
	mismatchIDs := collectMismatchIDs(result.Mismatches)
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
		executionEvidence = runAssistiveExecution(r.Context(), traceID, req, role, mismatchIDs, result.Mismatches)
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

func runAssistiveExecution(ctx context.Context, traceID string, req request.PresentationFindings, role stagepresentation.Role, mismatchIDs []string, mismatches []types.Mismatch) response.ExecutionEvidence {
	execCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	prompt := fmt.Sprintf("Generate assistive PMO-style risk context for %d mismatch(es) without overriding deterministic evidence.", len(mismatches))
	result, err := execution.LocalStubExecutor{}.Analyze(execCtx, execution.CodexRequest{
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
