package workspace

import (
	"context"
	"net/http"

	"context-os/apps/api/response"
	"context-os/domain/repository"
)

// AnalysisBasket handles GET/PUT /workspace/analysis-basket.
//
// @Summary      Read or save the workspace analysis basket
// @Description  Persists selected analysis evidence for one workspace.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        workspace_id  query     string  false  "Workspace path or ID"
// @Param        body          body      request.AnalysisBasket  false  "Analysis basket payload"
// @Success      200           {object}  request.AnalysisBasket
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      503           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /workspace/analysis-basket [get]
// @Router       /workspace/analysis-basket [put]
func (h *Handler) AnalysisBasket(w http.ResponseWriter, r *http.Request) {
	h.workspaceUIState(w, r, "analysis_basket", emptyAnalysisBasket, validateAnalysisBasketPayload)
}

// FindingActions handles GET/PUT /workspace/finding-actions.
//
// @Summary      Read or save the workspace finding action checklist
// @Description  Persists finding action statuses for one workspace.
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        workspace_id  query     string  false  "Workspace path or ID"
// @Param        body          body      request.FindingActions  false  "Finding actions payload"
// @Success      200           {object}  request.FindingActions
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      405           {object}  map[string]string
// @Failure      503           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /workspace/finding-actions [get]
// @Router       /workspace/finding-actions [put]
func (h *Handler) FindingActions(w http.ResponseWriter, r *http.Request) {
	h.workspaceUIState(w, r, "finding_actions", emptyFindingActions, validateFindingActionsPayload)
}

// workspaceUIState implements the shared GET/PUT flow for durable workspace UI state.
func (h *Handler) workspaceUIState(
	w http.ResponseWriter,
	r *http.Request,
	stateKey string,
	empty func(string) any,
	validate func([]byte) (string, []byte, error),
) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or PUT required")
		return
	}
	if h.uiState == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "state_unavailable", "workspace UI state is unavailable for this store")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), workspaceRequestTimeout)
	defer cancel()

	if r.Method == http.MethodGet {
		workspaceRef := workspaceRefFromQuery(r)
		if workspaceRef == "" {
			response.WriteError(w, http.StatusBadRequest, "invalid_request", "workspace_id query parameter is required")
			return
		}
		workspace, err := h.resolveWorkspace(ctx, workspaceRef)
		if err != nil {
			writeResolveWorkspaceError(w, err)
			return
		}
		state, err := h.uiState.Get(ctx, workspace.ID, stateKey)
		if err != nil {
			response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		if state == nil {
			response.WriteJSON(w, http.StatusOK, empty(workspace.Path))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(state.PayloadJSON)
		return
	}

	body, err := readBoundedBody(w, r, 1<<20)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	workspaceRef, payloadJSON, err := validate(body)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	workspace, err := h.resolveWorkspace(ctx, workspaceRef)
	if err != nil {
		writeResolveWorkspaceError(w, err)
		return
	}
	if err := h.uiState.Put(ctx, repository.WorkspaceUIState{
		WorkspaceID: workspace.ID,
		StateKey:    stateKey,
		PayloadJSON: payloadJSON,
	}); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payloadJSON)
}
