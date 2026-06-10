package googledrive

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

// NewConnector returns a Google Drive source connector that ingests Docs, Sheets, and Slides.
func NewConnector() contracts.MCPSourceConnector {
	return newConnector(&http.Client{Timeout: 15 * time.Second}, defaultDriveAPIBaseURL, defaultSlidesAPIBaseURL, sleepContext)
}

// newConnector wires the base MCP connector with injectable HTTP and sleep dependencies for tests.
func newConnector(client httpClient, driveAPIBaseURL, slidesAPIBaseURL string, sleep func(context.Context, time.Duration) error) connector {
	return connector{
		base:             source.NewMCPConnector("googledrive", contracts.CapabilityFiles),
		client:           client,
		driveAPIBaseURL:  strings.TrimRight(driveAPIBaseURL, "/"),
		slidesAPIBaseURL: strings.TrimRight(slidesAPIBaseURL, "/"),
		sleep:            sleep,
	}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

// Ingest lists supported Google Drive files in a folder, downloads their text content, and emits one event per file.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, c.connectorError(req, defaultFolderObjectType, "", contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
	}

	metadata := cloneMetadata(req.Metadata)
	if strings.TrimSpace(req.Content) != "" {
		req.Metadata = metadata
		return c.base.Ingest(ctx, req)
	}

	folderID, err := folderIDFromRequest(req.URI, metadata)
	if err != nil {
		return nil, c.connectorError(req, defaultFolderObjectType, "", contracts.ErrorKindInvalidRequest, false, err)
	}
	metadata[metadataFolderID] = folderID

	token, credentialType, err := c.accessToken(ctx, metadata)
	if err != nil {
		kind, retryable := classifyGoogleError(err)
		return nil, c.connectorError(req, defaultFolderObjectType, folderID, kind, retryable, err)
	}
	metadata[metadataCredentialTyp] = credentialType

	files, err := c.listFiles(ctx, folderID, req.Cursor, token)
	if err != nil {
		kind, retryable := classifyGoogleError(err)
		return nil, c.connectorError(req, defaultFolderObjectType, folderID, kind, retryable, err)
	}
	if len(files) == 0 {
		return nil, c.connectorError(req, defaultFolderObjectType, folderID, contracts.ErrorKindInvalidRequest, false, errors.New("google drive folder has no supported files"))
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].ID == files[j].ID {
			return files[i].ModifiedTime < files[j].ModifiedTime
		}
		return files[i].ID < files[j].ID
	})

	ingested := make([]events.Event, 0, len(files))
	for _, file := range files {
		event, eventErr := c.ingestFile(ctx, req, metadata, file, token)
		if eventErr != nil {
			kind, retryable := classifyGoogleError(eventErr)
			return nil, c.connectorError(req, defaultFileObjectType, file.ID, kind, retryable, eventErr)
		}
		ingested = append(ingested, event)
	}

	return ingested, nil
}

// ingestFile exports one supported Drive file, stamps provenance metadata, and delegates event creation to the base connector.
func (c connector) ingestFile(ctx context.Context, req contracts.SourceRequest, baseMetadata map[string]string, file driveFile, token string) (events.Event, error) {
	content, exportFormat, err := c.fileContent(ctx, file, token)
	if err != nil {
		return events.Event{}, err
	}

	metadata := cloneMetadata(baseMetadata)
	metadata[contracts.MetadataObjectType] = defaultFileObjectType
	metadata[contracts.MetadataObjectID] = file.ID
	metadata[events.MetadataSourceID] = "googledrive:file:" + file.ID
	metadata[events.MetadataEventID] = stableEventID(file.ID, file.ModifiedTime)
	metadata[metadataFileID] = file.ID
	metadata[metadataFileName] = file.Name
	metadata[metadataFileMimeType] = file.MimeType
	metadata[metadataModifiedTime] = file.ModifiedTime
	metadata[metadataExportFormat] = exportFormat
	metadata["url"] = driveFileURL(file.ID)

	baseReq := contracts.SourceRequest{
		URI:      driveFileURL(file.ID),
		Content:  content,
		Cursor:   file.ModifiedTime,
		Metadata: metadata,
	}
	created, err := c.base.Ingest(ctx, baseReq)
	if err != nil {
		return events.Event{}, err
	}
	if len(created) != 1 {
		return events.Event{}, errors.New("google drive connector expected one event per file")
	}
	return created[0], nil
}
