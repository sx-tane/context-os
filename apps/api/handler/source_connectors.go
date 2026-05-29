package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"context-os/apps/api/request"
	"context-os/apps/api/response"
	"context-os/domain/contracts"
	"context-os/domain/events"
	codexsource "context-os/internal/source/codex"
	filesystemsource "context-os/internal/source/filesystem"
	jirasource "context-os/internal/source/jira"
)

const sourceIngestTimeout = 20 * time.Second

type sourceIngestInput struct {
	URI      string
	Content  string
	Cursor   string
	Metadata map[string]string
}

type sourceIngestDecoder func(*json.Decoder) (sourceIngestInput, error)

// JiraStatus handles GET /jira/status.
// It reports whether Jira base URL and authentication environment variables are configured.
//
// @Summary      Jira connection status
// @Description  Returns whether Jira base URL and authentication environment variables are configured.
// @Tags         jira
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      405  {object}  map[string]string
// @Router       /jira/status [get]
func JiraStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required")
		return
	}

	baseURL := strings.TrimSpace(os.Getenv("JIRA_BASE_URL"))
	tokenConfigured := strings.TrimSpace(os.Getenv("JIRA_TOKEN")) != ""
	emailConfigured := strings.TrimSpace(os.Getenv("JIRA_EMAIL")) != ""

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":        baseURL != "",
		"base_url":         baseURL,
		"token_configured": tokenConfigured,
		"email_configured": emailConfigured,
	})
}

// JiraIngest handles POST /jira/ingest by ingesting a Jira issue or project
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a Jira issue or project
// @Description  Fetches Jira issue or project context by URI and returns a provenance-rich event.
// @Tags         jira
// @Accept       json
// @Produce      json
// @Param        body  body      request.JiraIngest  true  "Jira ingest request"
// @Success      200   {object}  response.JiraIngest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /jira/ingest [post]
func JiraIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	var req request.JiraIngest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	metadata := jiraMetadata(req)
	connector := jirasource.NewConnector()
	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		if strings.TrimSpace(req.URI) == "" {
			response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri is required for provider=codex")
			return
		}
		metadata = map[string]string{codexsource.MetadataPlugin: codexsource.PluginAtlassianRovo}
		connector = codexsource.NewConnector()
	}

	writeSourceIngest(w, r, connector, sourceIngestInput{
		URI:      req.URI,
		Content:  req.Content,
		Cursor:   req.Cursor,
		Metadata: metadata,
	})
}

// FilesystemIngest handles POST /filesystem/ingest by ingesting local files
// via the MCP source connector and returning a provenance-rich event summary.
//
// @Summary      Ingest a local filesystem artifact
// @Description  Reads a local file or folder path and returns provenance-rich events with extracted text and file metadata.
// @Tags         filesystem
// @Accept       json
// @Produce      json
// @Param        body  body      request.FilesystemIngest  true  "Filesystem ingest request"
// @Success      200   {object}  response.FilesystemIngest
// @Failure      400   {object}  map[string]string
// @Failure      405   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Failure      503   {object}  map[string]string
// @Router       /filesystem/ingest [post]
func FilesystemIngest(w http.ResponseWriter, r *http.Request) {
	runSourceIngest(w, r, filesystemsource.NewConnector(), decodeFilesystemIngest)
}

func runSourceIngest(w http.ResponseWriter, r *http.Request, connector contracts.MCPSourceConnector, decode sourceIngestDecoder) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	input, err := decode(json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	writeSourceIngest(w, r, connector, input)
}

func writeSourceIngest(w http.ResponseWriter, r *http.Request, connector contracts.MCPSourceConnector, input sourceIngestInput) {
	uri := strings.TrimSpace(input.URI)
	if uri == "" && strings.TrimSpace(input.Content) == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "uri or content is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sourceIngestTimeout)
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
	return newIngestResponse(connector.Name(), capabilityStrings(connector.Capabilities()), ingested)
}

func newIngestResponse(connectorName string, capabilities []string, ingested []events.Event) response.Ingest {
	previews := make([]string, len(ingested))
	metadataItems := make([]map[string]string, len(ingested))
	for index, event := range ingested {
		previews[index] = preview(event.Content)
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

func decodeFilesystemIngest(dec *json.Decoder) (sourceIngestInput, error) {
	var req request.FilesystemIngest
	if err := dec.Decode(&req); err != nil {
		return sourceIngestInput{}, err
	}
	metadata := cloneStringMap(req.Metadata)
	setMetadata(metadata, "filesystem_include", req.Include)
	setMetadata(metadata, "filesystem_exclude", req.Exclude)
	return sourceIngestInput{URI: req.URI, Content: req.Content, Cursor: req.Cursor, Metadata: metadata}, nil
}

func capabilityStrings(capabilities []contracts.Capability) []string {
	out := make([]string, len(capabilities))
	for i, capability := range capabilities {
		out[i] = string(capability)
	}
	return out
}

func cloneStringMap(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func setMetadata(metadata map[string]string, key string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	metadata[key] = trimmed
}

func jiraMetadata(req request.JiraIngest) map[string]string {
	metadata := cloneStringMap(req.Metadata)
	setMetadata(metadata, "jira_token", req.Token)
	setMetadata(metadata, "jira_email", req.Email)
	setMetadata(metadata, "jira_api_base_url", req.APIBaseURL)
	setMetadata(metadata, "jira_expand", req.Expand)
	return metadata
}
