// Package handler contains HTTP handlers for the ContextOS API.
package handler

import (
	"net/http"

	"context-os/apps/api/response"
)

// Health responds to GET /health with a JSON liveness payload.
//
// @Summary      Liveness check
// @Description  Returns ok when the API process is running.
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func Health(w http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "context-os-api",
	})
}
