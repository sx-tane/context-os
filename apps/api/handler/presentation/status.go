package presentation

import (
	"context-os/apps/api/response"
	"net/http"
)

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
