package googledrive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// fileContent exports supported Drive file content into text the base connector can ingest.
func (c connector) fileContent(ctx context.Context, file driveFile, token string) (string, string, error) {
	switch file.MimeType {
	case googleDocsMimeType:
		body, _, err := c.get(ctx, c.driveAPIBaseURL+"/files/"+url.PathEscape(file.ID)+"/export?mimeType="+url.QueryEscape(googleDocsExportMimeType), token)
		if err != nil {
			return "", "", err
		}
		return string(body), googleDocsExportMimeType, nil
	case googleSheetsMimeType:
		body, _, err := c.get(ctx, c.driveAPIBaseURL+"/files/"+url.PathEscape(file.ID)+"/export?mimeType="+url.QueryEscape(googleSheetsExportMimeType), token)
		if err != nil {
			return "", "", err
		}
		formatted, formatErr := formatCSVAsTable(body)
		if formatErr != nil {
			return "", "", formatErr
		}
		return formatted, googleSheetsExportMimeType, nil
	case googleSlidesMimeType:
		body, _, err := c.get(ctx, c.slidesAPIBaseURL+"/presentations/"+url.PathEscape(file.ID), token)
		if err != nil {
			return "", "", err
		}
		formatted, formatErr := formatSlidesAsText(body)
		if formatErr != nil {
			return "", "", formatErr
		}
		return formatted, "slides:text", nil
	default:
		return "", "", fmt.Errorf("unsupported google drive mime type %q", file.MimeType)
	}
}

// listFiles pages through Drive folder search results for supported Docs, Sheets, and Slides files.
func (c connector) listFiles(ctx context.Context, folderID, cursor, token string) ([]driveFile, error) {
	query := []string{
		fmt.Sprintf("'%s' in parents", escapeDriveQuery(folderID)),
		"trashed = false",
		"(mimeType = '" + googleDocsMimeType + "' or mimeType = '" + googleSheetsMimeType + "' or mimeType = '" + googleSlidesMimeType + "')",
	}
	if trimmedCursor := strings.TrimSpace(cursor); trimmedCursor != "" {
		parsedCursor, err := time.Parse(time.RFC3339Nano, trimmedCursor)
		if err != nil {
			return nil, fmt.Errorf("parse cursor: %w", err)
		}
		query = append(query, fmt.Sprintf("modifiedTime > '%s'", parsedCursor.UTC().Format(time.RFC3339Nano)))
	}

	params := url.Values{}
	params.Set("fields", "nextPageToken,files(id,name,mimeType,modifiedTime)")
	params.Set("includeItemsFromAllDrives", "true")
	params.Set("supportsAllDrives", "true")
	params.Set("pageSize", "1000")
	params.Set("q", strings.Join(query, " and "))
	params.Set("orderBy", "modifiedTime,name")

	files := make([]driveFile, 0)
	nextPageToken := ""
	for {
		pageParams := url.Values{}
		for key, values := range params {
			copied := make([]string, len(values))
			copy(copied, values)
			pageParams[key] = copied
		}
		if nextPageToken != "" {
			pageParams.Set("pageToken", nextPageToken)
		}

		body, _, err := c.get(ctx, c.driveAPIBaseURL+"/files?"+pageParams.Encode(), token)
		if err != nil {
			return nil, err
		}

		var payload struct {
			NextPageToken string      `json:"nextPageToken"`
			Files         []driveFile `json:"files"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode drive file list: %w", err)
		}
		files = append(files, payload.Files...)
		if strings.TrimSpace(payload.NextPageToken) == "" {
			return files, nil
		}
		nextPageToken = payload.NextPageToken
	}
}

// get performs an authenticated Google API GET request with retry behavior.
func (c connector) get(ctx context.Context, endpoint, token string) ([]byte, http.Header, error) {
	return c.doRequest(ctx, http.MethodGet, endpoint, token, "", "")
}

// postForm performs a Google API form POST request with retry behavior.
func (c connector) postForm(ctx context.Context, endpoint, body string) ([]byte, http.Header, error) {
	return c.doRequest(ctx, http.MethodPost, endpoint, "", "application/x-www-form-urlencoded", body)
}

// doRequest sends one Google API request with bounded response reads and retry/backoff for transient failures.
func (c connector) doRequest(ctx context.Context, method, endpoint, token, contentType, body string) ([]byte, http.Header, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		var requestBody io.Reader
		if body != "" {
			requestBody = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, requestBody)
		if err != nil {
			return nil, nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt == maxAttempts {
				return nil, nil, lastErr
			}
			if sleepErr := c.sleep(ctx, backoffDuration(attempt, nil)); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}

		responseBody, readErr := readResponseBody(resp)
		closeErr := resp.Body.Close()
		if readErr != nil {
			if closeErr != nil {
				return nil, nil, errors.Join(readErr, closeErr)
			}
			return nil, nil, readErr
		}
		if closeErr != nil {
			return nil, nil, closeErr
		}

		lowerBody := strings.ToLower(string(responseBody))
		isRateLimit403 := resp.StatusCode == http.StatusForbidden &&
			(strings.Contains(lowerBody, "ratelimitexceeded") || strings.Contains(lowerBody, "userratelimitexceeded"))
		if resp.StatusCode == http.StatusTooManyRequests || isRateLimit403 || resp.StatusCode >= http.StatusInternalServerError {
			lastErr = googleAPIError{status: resp.StatusCode, message: strings.TrimSpace(string(responseBody))}
			if attempt == maxAttempts {
				return nil, resp.Header.Clone(), lastErr
			}
			if sleepErr := c.sleep(ctx, backoffDuration(attempt, resp.Header)); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}

		if resp.StatusCode >= http.StatusBadRequest {
			return nil, resp.Header.Clone(), googleAPIError{status: resp.StatusCode, message: strings.TrimSpace(string(responseBody))}
		}
		return responseBody, resp.Header.Clone(), nil
	}

	if lastErr == nil {
		lastErr = errors.New("request failed without an explicit error")
	}
	return nil, nil, lastErr
}
