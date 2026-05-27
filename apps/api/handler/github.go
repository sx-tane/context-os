package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	githubsource "context-os/internal/source/github"
)

// ingestTimeout caps a single GitHub connector call so the HTTP handler stays responsive.
const ingestTimeout = 20 * time.Second

// GithubIngest handles POST /github/ingest by ingesting a single GitHub artifact
// via the MCP source connector and returning a provenance-rich event summary.
func GithubIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.GithubIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	uri := strings.TrimSpace(req.URI)
	if uri == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri is required")
		return
	}

	metadata := map[string]string{}
	if token := strings.TrimSpace(req.Token); token != "" {
		metadata["github_token"] = token
	}

	ctx, cancel := context.WithTimeout(r.Context(), ingestTimeout)
	defer cancel()

	connector := githubsource.NewConnector()
	ingested, err := connector.Ingest(ctx, contracts.SourceRequest{URI: uri, Metadata: metadata})
	if err != nil {
		response.WriteConnectorError(w, err)
		return
	}
	if len(ingested) == 0 {
		response.WriteError(w, http.StatusInternalServerError, "empty_result", "connector returned no events")
		return
	}

	event := ingested[0]
	caps := connector.Capabilities()
	capStrings := make([]string, len(caps))
	for i, c := range caps {
		capStrings[i] = string(c)
	}

	response.WriteJSON(w, http.StatusOK, response.GithubIngest{
		Connector:    connector.Name(),
		Capabilities: capStrings,
		Event:        event,
		Preview:      event.Content,
		Metadata:     event.Metadata,
	})
}
