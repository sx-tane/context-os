// Package chat provides HTTP handlers for local workspace chat queries.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"context-os/apps/api/handler/shared"
	"context-os/apps/api/request"
	"context-os/apps/api/response"
	internalchat "context-os/internal/chat"
)

const chatRequestTimeout = 5 * time.Minute

// Handler holds the service dependency for local chat handlers.
type Handler struct {
	service       *internalchat.Service
	evidenceSaver EvidenceSaver
}

// NewHandler returns a Handler wired to the provided chat service.
func NewHandler(service *internalchat.Service) *Handler {
	return &Handler{service: service, evidenceSaver: persistentEvidenceSaver{}}
}

// WithEvidenceSaver replaces the default live evidence persistence adapter.
func (h *Handler) WithEvidenceSaver(saver EvidenceSaver) *Handler {
	h.evidenceSaver = saver
	return h
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

	chatResult := mapChatResult(result)
	chatResult = h.startAsyncEvidenceSave(ctx, chatResult)
	response.WriteJSON(w, http.StatusOK, chatResult)
}

// StreamQuery handles POST /chat/query/stream.
//
// @Summary      Stream workspace context query
// @Description  Streams chat query progress, including live Codex logs, then emits the final chat result.
// @Tags         chat
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body      request.ChatQuery  true  "Chat query"
// @Success      200
// @Failure      405   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /chat/query/stream [post]
func (h *Handler) StreamQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	if h.service == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "store_unavailable", "chat store is unavailable")
		return
	}

	f, ok := shared.SSEHeaders(w)
	if !ok {
		return
	}
	sw := shared.NewSSEWriter(w, f)

	var req request.ChatQuery
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
		sw.Error("invalid_json", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), chatRequestTimeout)
	defer cancel()

	type streamResult struct {
		result internalchat.Result
		err    error
	}
	resultCh := make(chan streamResult, 1)
	started := time.Now()
	go func() {
		result, err := h.service.Query(ctx, internalchat.Query{
			WorkspaceID:   req.WorkspaceID,
			WorkspacePath: req.WorkspacePath,
			Message:       req.Message,
			Connector:     req.Connector,
			SourceURI:     req.SourceURI,
			Timezone:      req.Timezone,
			LocalDate:     req.LocalDate,
			Limit:         req.Limit,
			Progress:      sw.Log,
		})
		resultCh <- streamResult{result: result, err: err}
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case result := <-resultCh:
			if result.err != nil {
				writeStreamQueryError(sw, result.err)
				return
			}
			chatResult := mapChatResult(result.result)
			if shouldSaveEvidence(chatResult) {
				chatResult.EvidenceSaveStatus = evidenceStatusSaving
				sw.Event("answer", mustJSON(chatResult))
				chatResult = h.saveEvidence(ctx, chatResult, sw.Log)
			}
			sw.Result(chatResult)
			return
		case <-ctx.Done():
			sw.Error("query_timeout", ctx.Err().Error())
			return
		case <-ticker.C:
			elapsed := int(time.Since(started).Seconds())
			body, _ := json.Marshal(map[string]any{"elapsed": elapsed, "status": "running"})
			sw.Event("status", string(body))
		}
	}
}

func (h *Handler) startAsyncEvidenceSave(ctx context.Context, result response.ChatQuery) response.ChatQuery {
	if !shouldSaveEvidence(result) {
		return result
	}
	result.EvidenceSaveStatus = evidenceStatusSaving
	go func() {
		_ = h.saveEvidence(context.WithoutCancel(ctx), result, nil)
	}()
	return result
}

func (h *Handler) saveEvidence(ctx context.Context, result response.ChatQuery, progress func(string)) response.ChatQuery {
	input, ok := evidenceSaveInput(result)
	if !ok {
		result.EvidenceSaveStatus = evidenceStatusSkipped
		return result
	}
	if h.evidenceSaver == nil {
		result.EvidenceSaveStatus = evidenceStatusSkipped
		return result
	}
	saved, err := h.evidenceSaver.Save(ctx, input, progress)
	if err != nil {
		result.EvidenceSaveStatus = evidenceStatusError
		result.EvidenceSaveError = err.Error()
		if progress != nil {
			progress("• Local DB evidence save failed: " + err.Error())
		}
		return result
	}
	result.EvidenceSaveStatus = evidenceStatusSaved
	result.EvidenceEventCount = saved.EventCount
	if saved.GraphUpdated {
		result.EvidenceGraphStatus = "updated"
	} else {
		result.EvidenceGraphStatus = "unchanged"
	}
	result.EvidenceGraphEntityCount = saved.EntityCount
	result.EvidenceGraphRelationshipCount = saved.RelationshipCount
	return result
}

func shouldSaveEvidence(result response.ChatQuery) bool {
	_, ok := evidenceSaveInput(result)
	return ok
}

func mustJSON(value any) string {
	body, _ := json.Marshal(value)
	return string(body)
}

func writeStreamQueryError(sw *shared.SSEWriter, err error) {
	switch {
	case errors.Is(err, internalchat.ErrWorkspaceRequired), errors.Is(err, internalchat.ErrMessageRequired):
		sw.Error("invalid_request", err.Error())
	case errors.Is(err, internalchat.ErrWorkspaceNotFound):
		sw.Error("not_found", err.Error())
	default:
		sw.Error("query_error", err.Error())
	}
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
		Intent:             result.Intent,
		WorkspaceID:        result.WorkspaceID,
		WorkspacePath:      result.WorkspacePath,
		Connector:          result.Connector,
		SourceURI:          result.SourceURI,
		Provider:           result.Provider,
		Answer:             result.Answer,
		Summary:            result.Summary,
		AnswerSections:     mapAnswerSections(result.AnswerSections),
		RangeStart:         rangeStart,
		RangeEnd:           rangeEnd,
		ArtifactCount:      len(result.Artifacts),
		Artifacts:          response.NewArtifacts(result.Artifacts),
		Syncs:              result.Syncs,
		EvidenceSaveStatus: evidenceStatusSkipped,
	}
}

func mapAnswerSections(sections []internalchat.AnswerSection) []response.AnswerSection {
	out := make([]response.AnswerSection, 0, len(sections))
	for _, section := range sections {
		out = append(out, response.AnswerSection{
			SourceLabel: section.SourceLabel,
			Connector:   section.Connector,
			SourceURI:   section.SourceURI,
			Summary:     section.Summary,
			Facts:       append([]string(nil), section.Facts...),
			OpenItems:   append([]string(nil), section.OpenItems...),
			CodingNotes: append([]string(nil), section.CodingNotes...),
			Links:       append([]string(nil), section.Links...),
			Timestamps:  append([]string(nil), section.Timestamps...),
			Confidence:  section.Confidence,
			Status:      section.Status,
		})
	}
	return out
}
