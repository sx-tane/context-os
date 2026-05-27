// Package handler contains HTTP handlers for the ContextOS API.
package handler

import (
	"net/http"

	"context-os/apps/api/response"
)

// Health responds to GET /health with a JSON liveness payload.
func Health(w http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "context-os-api",
	})
}
