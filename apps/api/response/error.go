// Package response provides shared JSON response helpers and error types for the ContextOS API.
package response

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"context-os/domain/contracts"
)

// WriteJSON encodes v as JSON with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

// WriteError emits a compact error envelope with a machine-readable code and human message.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]string{"error": code, "message": message})
}

// WriteConnectorError maps a structured ConnectorError to an appropriate HTTP response.
func WriteConnectorError(w http.ResponseWriter, err error) {
	var connectorErr *contracts.ConnectorError
	if !errors.As(err, &connectorErr) {
		WriteError(w, http.StatusInternalServerError, "connector_error", err.Error())
		return
	}

	status := http.StatusBadGateway
	switch connectorErr.Kind {
	case contracts.ErrorKindInvalidRequest:
		status = http.StatusBadRequest
	case contracts.ErrorKindCanceled:
		status = http.StatusGatewayTimeout
	case contracts.ErrorKindPermanent:
		status = http.StatusBadGateway
	case contracts.ErrorKindTemporary:
		status = http.StatusServiceUnavailable
	}

	WriteJSON(w, status, map[string]any{
		"error":       string(connectorErr.Kind),
		"message":     connectorErr.Error(),
		"connector":   connectorErr.Connector,
		"uri":         connectorErr.URI,
		"object_type": connectorErr.ObjectType,
		"object_id":   connectorErr.ObjectID,
		"retryable":   connectorErr.Retryable,
	})
}
