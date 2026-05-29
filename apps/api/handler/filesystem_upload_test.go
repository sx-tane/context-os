package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"context-os/apps/api/response"
)

func TestFilesystemUploadIngestsSingleFile(t *testing.T) {
	t.Setenv("FILESYSTEM_UPLOAD_ROOT", t.TempDir())

	recorder := httptest.NewRecorder()
	request := newMultipartUploadRequest(t, map[string]string{
		"outside.md": "uploaded requirement\n",
	})

	FilesystemUpload(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	got := decodeIngestResponse(t, recorder)
	if got.EventCount != 1 {
		t.Fatalf("event_count = %d, want 1", got.EventCount)
	}
	if got.Metadata["filesystem_upload_id"] == "" || got.Metadata["filesystem_upload_original_name"] != "outside.md" {
		t.Fatalf("missing upload metadata: %#v", got.Metadata)
	}
	if got.Metadata["filesystem_format"] != "text" {
		t.Fatalf("filesystem_format = %q, want text", got.Metadata["filesystem_format"])
	}
	if _, err := os.Stat(got.Metadata["path"]); err != nil {
		t.Fatalf("expected staged file to exist: %v", err)
	}
	if !strings.Contains(got.Preview, "uploaded requirement") {
		t.Fatalf("preview = %q, want uploaded content", got.Preview)
	}
}

func TestFilesystemUploadIngestsFolderPaths(t *testing.T) {
	t.Setenv("FILESYSTEM_UPLOAD_ROOT", t.TempDir())

	recorder := httptest.NewRecorder()
	request := newMultipartUploadRequest(t, map[string]string{
		"folder/a.md":        "first upload\n",
		"folder/nested/b.md": "second upload\n",
	})

	FilesystemUpload(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	got := decodeIngestResponse(t, recorder)
	if got.EventCount != 2 {
		t.Fatalf("event_count = %d, want 2", got.EventCount)
	}
	if got.Metadata["filesystem_ingest_mode"] != "folder" {
		t.Fatalf("filesystem_ingest_mode = %q, want folder", got.Metadata["filesystem_ingest_mode"])
	}
	if got.Metadata["filesystem_relative_path"] != "folder/a.md" {
		t.Fatalf("filesystem_relative_path = %q, want folder/a.md", got.Metadata["filesystem_relative_path"])
	}
	if got.Metadata["filesystem_upload_file_count"] != "2" {
		t.Fatalf("filesystem_upload_file_count = %q, want 2", got.Metadata["filesystem_upload_file_count"])
	}
	if filepath.Base(filepath.Dir(got.Metadata["filesystem_upload_root"])) != filepath.Base(os.Getenv("FILESYSTEM_UPLOAD_ROOT")) {
		t.Fatalf("unexpected upload root metadata: %#v", got.Metadata)
	}
}

func TestFilesystemUploadRejectsTraversalPath(t *testing.T) {
	t.Setenv("FILESYSTEM_UPLOAD_ROOT", t.TempDir())

	recorder := httptest.NewRecorder()
	request := newMultipartUploadRequest(t, map[string]string{
		"../evil.md": "nope\n",
	})

	FilesystemUpload(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "invalid_upload_path") {
		t.Fatalf("expected invalid_upload_path response, got %s", recorder.Body.String())
	}
}

func newMultipartUploadRequest(t *testing.T, files map[string]string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for filename, content := range files {
		part, err := writer.CreateFormFile("files", filename)
		if err != nil {
			t.Fatalf("create multipart file: %v", err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("write multipart file: %v", err)
		}
		if err := writer.WriteField("paths", filename); err != nil {
			t.Fatalf("write multipart path: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/filesystem/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func decodeIngestResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Ingest {
	t.Helper()

	var got response.Ingest
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return got
}
