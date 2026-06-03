// Package chat provides HTTP handlers for local workspace chat queries.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	internalchat "context-os/internal/chat"
)

const chatRequestTimeout = 150 * time.Second

// Handler holds the service dependency for local chat handlers.
type Handler struct {
	service *internalchat.Service
}

// NewHandler returns a Handler wired to the provided chat service.
func NewHandler(service *internalchat.Service) *Handler {
	return &Handler{service: service}
}

// Query handles POST /chat/query.
//
// @Summary      Query workspace context
// @Description  Answers source, status, and findings-intent questions from local artifacts, with optional Codex live lookup for configured source-specific questions.
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        body  body      request.ChatQuery  true  "Chat query"
// @Success      200   {object}  response.ChatQuery
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /chat/query [post]
func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	if h.service == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "store_unavailable", "chat store is unavailable")
		return
	}

	var req request.ChatQuery
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), chatRequestTimeout)
	defer cancel()

	result, err := h.service.Query(ctx, internalchat.Query{
		WorkspaceID:   req.WorkspaceID,
		WorkspacePath: req.WorkspacePath,
		Message:       req.Message,
		Connector:     req.Connector,
		SourceURI:     req.SourceURI,
		Timezone:      req.Timezone,
		LocalDate:     req.LocalDate,
		Limit:         req.Limit,
	})
	if err != nil {
		writeQueryError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, mapChatResult(result))
}

func writeQueryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, internalchat.ErrWorkspaceRequired), errors.Is(err, internalchat.ErrMessageRequired):
		response.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, internalchat.ErrWorkspaceNotFound):
		response.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		response.WriteError(w, http.StatusInternalServerError, "query_error", err.Error())
	}
}

func mapChatResult(result internalchat.Result) response.ChatQuery {
	var rangeStart, rangeEnd string
	if result.Since != nil {
		rangeStart = result.Since.Format(time.RFC3339)
	}
	if result.Until != nil {
		rangeEnd = result.Until.Format(time.RFC3339)
	}

	return response.ChatQuery{
		Intent:        result.Intent,
		WorkspaceID:   result.WorkspaceID,
		WorkspacePath: result.WorkspacePath,
		Connector:     result.Connector,
		SourceURI:     result.SourceURI,
		Provider:      result.Provider,
		Answer:        result.Answer,
		Summary:       result.Summary,
		RangeStart:    rangeStart,
		RangeEnd:      rangeEnd,
		ArtifactCount: len(result.Artifacts),
		Artifacts:     response.NewArtifacts(result.Artifacts),
		Syncs:         result.Syncs,
	}
}
