package googledrive

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// folderIDFromRequest resolves the Drive folder ID from metadata, environment, direct IDs, or supported URI shapes.
func folderIDFromRequest(uri string, metadata map[string]string) (string, error) {
	if folderID := strings.TrimSpace(metadata[metadataFolderID]); folderID != "" {
		return folderID, nil
	}
	if folderID := strings.TrimSpace(os.Getenv(googleDriveFolderIDEnv)); folderID != "" && strings.TrimSpace(uri) == "" {
		return folderID, nil
	}

	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return "", errors.New("google drive folder uri is required")
	}
	if !strings.Contains(trimmed, "://") && !strings.Contains(trimmed, "/") {
		return trimmed, nil
	}

	if strings.HasPrefix(trimmed, "googledrive://") || strings.HasPrefix(trimmed, "gdrive://") {
		trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "googledrive://"), "gdrive://")
		parts := splitPath(trimmed)
		if len(parts) >= 2 && parts[0] == "folder" {
			return parts[1], nil
		}
		if len(parts) == 1 {
			return parts[0], nil
		}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse google drive uri: %w", err)
	}
	if id := strings.TrimSpace(parsed.Query().Get("id")); id != "" {
		return id, nil
	}
	if !strings.EqualFold(parsed.Host, "drive.google.com") {
		return "", errors.New("google drive uri must point to drive.google.com or use googledrive://folder/<id>")
	}

	parts := splitPath(parsed.Path)
	for index, part := range parts {
		if part == "folders" && index+1 < len(parts) {
			return parts[index+1], nil
		}
	}
	return "", errors.New("google drive folder id not found in uri")
}

// driveFileURL returns the stable browser URL stored in ingested file metadata.
func driveFileURL(fileID string) string {
	return "https://drive.google.com/file/d/" + url.PathEscape(fileID) + "/view"
}

// stableEventID creates replay-stable event IDs from a Drive file ID and modified timestamp.
func stableEventID(fileID, modifiedTime string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(fileID) + "\x00" + strings.TrimSpace(modifiedTime)))
	return "event:" + hex.EncodeToString(sum[:])
}

// cloneMetadata copies request metadata before the connector adds Drive-specific provenance keys.
func cloneMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

// escapeDriveQuery escapes quotes for Drive query string literals.
func escapeDriveQuery(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

// firstNonEmpty returns the first trimmed non-empty value from a precedence list.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// splitPath returns non-empty path segments after trimming leading and trailing slashes.
func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
