// Package shared provides HTTP handler plumbing shared across all ContextOS API domain handlers.
// It is an internal implementation package — callers outside apps/api should not import it.
package shared

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"context-os/apps/api/response"
	"context-os/domain/contracts"
	"context-os/domain/events"
)

// SourceIngestTimeout is the maximum duration allowed for a synchronous source ingest call.
const SourceIngestTimeout = 20 * time.Second

// SourceIngestInput carries the decoded fields for a source ingest request.
type SourceIngestInput struct {
	URI      string
	Content  string
	Cursor   string
	Metadata map[string]string
}

// SourceIngestDecoder decodes an HTTP request body into a SourceIngestInput.
type SourceIngestDecoder func(*json.Decoder) (SourceIngestInput, error)

// RunSourceIngest enforces the POST method guard, decodes the JSON body using decode,
// and delegates to WriteSourceIngest.
func RunSourceIngest(w http.ResponseWriter, r *http.Request, connector contracts.MCPSourceConnector, decode SourceIngestDecoder) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	input, err := decode(json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	WriteSourceIngest(w, r, connector, input)
}

// WriteSourceIngest validates the URI/content pair, calls connector.Ingest, and writes
// a JSON response.  It is the shared ingest path used by all domain handlers.
func WriteSourceIngest(w http.ResponseWriter, r *http.Request, connector contracts.MCPSourceConnector, input SourceIngestInput) {
	uri := strings.TrimSpace(input.URI)
	if uri == "" && strings.TrimSpace(input.Content) == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri or content is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), SourceIngestTimeout)
	defer cancel()

	ingested, err := connector.Ingest(ctx, contracts.SourceRequest{
		URI:      uri,
		Content:  input.Content,
		Cursor:   strings.TrimSpace(input.Cursor),
		Metadata: input.Metadata,
	})
	if err != nil {
		response.WriteConnectorError(w, err)
		return
	}
	if len(ingested) == 0 {
		response.WriteError(w, http.StatusInternalServerError, "empty_result", "connector returned no events")
		return
	}

	response.WriteJSON(w, http.StatusOK, sourceIngestResponse(connector, ingested))
}

func sourceIngestResponse(connector contracts.MCPSourceConnector, ingested []events.Event) response.Ingest {
	return NewIngestResponse(connector.Name(), CapabilityStrings(connector.Capabilities()), ingested)
}

// NewIngestResponse builds a response.Ingest from a connector name, capability list,
// and ingested events slice.
func NewIngestResponse(connectorName string, capabilities []string, ingested []events.Event) response.Ingest {
	previews := make([]string, len(ingested))
	metadataItems := make([]map[string]string, len(ingested))
	for index, event := range ingested {
		previews[index] = Preview(event.Content)
		metadataItems[index] = event.Metadata
	}

	return response.Ingest{
		Connector:     connectorName,
		Capabilities:  capabilities,
		Event:         ingested[0],
		Events:        ingested,
		EventCount:    len(ingested),
		Preview:       previews[0],
		Previews:      previews,
		Metadata:      ingested[0].Metadata,
		MetadataItems: metadataItems,
	}
}

// CapabilityStrings converts a slice of contracts.Capability to a slice of plain strings.
func CapabilityStrings(capabilities []contracts.Capability) []string {
	out := make([]string, len(capabilities))
	for i, capability := range capabilities {
		out[i] = string(capability)
	}
	return out
}

// CloneStringMap copies a metadata map, dropping keys with empty or whitespace-only values.
func CloneStringMap(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out[key] = value
	}
	return out
}

// SetMetadata sets key to the trimmed value in metadata if the trimmed value is non-empty.
func SetMetadata(metadata map[string]string, key string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	metadata[key] = trimmed
}

// NewSourceRequest constructs a contracts.SourceRequest with the given URI and metadata.
func NewSourceRequest(uri string, metadata map[string]string) contracts.SourceRequest {
	return contracts.SourceRequest{URI: uri, Metadata: metadata}
}

// Preview returns the first 500 runes of content as a display preview,
// appending "…" when the input exceeds that length.
func Preview(content string) string {
	runes := []rune(content)
	if len(runes) > 500 {
		return string(runes[:500]) + "…"
	}
	return content
}
