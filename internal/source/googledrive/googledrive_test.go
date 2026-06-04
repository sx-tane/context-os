package googledrive

// These tests use the internal package so they can build connectors with fake endpoints and backoff hooks.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
)

// TestIngestWithOAuthCredentialsPath verifies authorized-user OAuth credentials can list and ingest Docs, Sheets, and Slides with replay-stable IDs.
func TestIngestWithOAuthCredentialsPath(t *testing.T) {
	server := newGoogleDriveTestServer(t)
	connector := newConnector(server.Client(), server.URL+"/drive/v3", server.URL+"/slides/v1", func(context.Context, time.Duration) error { return nil })
	credentialsPath := writeAuthorizedUserCredentials(t, server.URL+"/token")

	req := contracts.SourceRequest{
		URI: "https://drive.google.com/drive/folders/folder-123",
		Metadata: map[string]string{
			MetadataOAuthCredentialsPath: credentialsPath,
		},
	}

	ingested, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(ingested) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(ingested))
	}

	byID := make(map[string]string, len(ingested))
	for _, event := range ingested {
		fileID := event.Metadata[metadataFileID]
		if event.Metadata["url"] != driveFileURL(fileID) {
			t.Fatalf("url = %q, want %q", event.Metadata["url"], driveFileURL(fileID))
		}
		if event.ID == "" {
			t.Fatalf("event id is empty for file %q", fileID)
		}
		byID[fileID] = event.ID
	}
	if got := findContent(ingested, "doc-1"); got != "Doc body" {
		t.Fatalf("doc content = %q, want %q", got, "Doc body")
	}
	if got := findContent(ingested, "sheet-1"); got != "name\tvalue\nalpha\t1\nbeta\t2" {
		t.Fatalf("sheet content = %q, want %q", got, "name\tvalue\nalpha\t1\nbeta\t2")
	}
	if got := findContent(ingested, "slide-1"); got != "Slide 1\nTitle\nBullet one\n\nSlide 2\nFollow-up" {
		t.Fatalf("slide content = %q, want %q", got, "Slide 1\nTitle\nBullet one\n\nSlide 2\nFollow-up")
	}

	replayed, err := connector.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("Ingest() replay error = %v", err)
	}
	for _, event := range replayed {
		fileID := event.Metadata[metadataFileID]
		if event.ID != byID[fileID] {
			t.Fatalf("replay event id for %q = %q, want %q", fileID, event.ID, byID[fileID])
		}
	}
}

// TestIngestWithServiceAccountCredentialsPath verifies service-account credentials can authenticate and ingest a Google Drive folder.
func TestIngestWithServiceAccountCredentialsPath(t *testing.T) {
	server := newGoogleDriveTestServer(t)
	connector := newConnector(server.Client(), server.URL+"/drive/v3", server.URL+"/slides/v1", func(context.Context, time.Duration) error { return nil })
	credentialsPath := writeServiceAccountCredentials(t, server.URL+"/token")

	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "googledrive://folder/folder-123",
		Metadata: map[string]string{
			MetadataServiceAccountPath: credentialsPath,
		},
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(ingested) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(ingested))
	}
	if ingested[0].Metadata[metadataCredentialTyp] != "service_account" {
		t.Fatalf("credential type = %q, want %q", ingested[0].Metadata[metadataCredentialTyp], "service_account")
	}
}

// TestIngestBacksOffOnRateLimit verifies Google Drive 429 responses are retried with backoff before succeeding.
func TestIngestBacksOffOnRateLimit(t *testing.T) {
	server := newGoogleDriveTestServer(t)
	server.rateLimitList.Store(true)

	var sleepCalls int32
	connector := newConnector(server.Client(), server.URL+"/drive/v3", server.URL+"/slides/v1", func(context.Context, time.Duration) error {
		atomic.AddInt32(&sleepCalls, 1)
		return nil
	})
	credentialsPath := writeAuthorizedUserCredentials(t, server.URL+"/token")

	ingested, err := connector.Ingest(context.Background(), contracts.SourceRequest{
		URI: "https://drive.google.com/drive/folders/folder-123",
		Metadata: map[string]string{
			MetadataOAuthCredentialsPath: credentialsPath,
		},
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if len(ingested) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(ingested))
	}
	if atomic.LoadInt32(&sleepCalls) == 0 {
		t.Fatalf("sleepCalls = %d, want > 0", sleepCalls)
	}
	if got := server.listCalls.Load(); got < 2 {
		t.Fatalf("list calls = %d, want at least 2", got)
	}
}

type googleDriveTestServer struct {
	*httptest.Server
	listCalls     atomic.Int32
	rateLimitList atomic.Bool
}

// newGoogleDriveTestServer returns a fake Google API server for connector tests.
func newGoogleDriveTestServer(t *testing.T) *googleDriveTestServer {
	t.Helper()

	server := &googleDriveTestServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token"}`))
	})
	mux.HandleFunc("/drive/v3/files", func(w http.ResponseWriter, r *http.Request) {
		call := server.listCalls.Add(1)
		if call == 1 && server.rateLimitList.Load() {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		if r.URL.Query().Get("q") == "" {
			t.Fatalf("q = %q, want non-empty", r.URL.Query().Get("q"))
		}
		if got := r.URL.Query().Get("orderBy"); got != "modifiedTime,name" {
			t.Fatalf("orderBy = %q, want %q", got, "modifiedTime,name")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"files":[{"id":"doc-1","name":"Doc","mimeType":"application/vnd.google-apps.document","modifiedTime":"2026-05-29T12:00:00Z"},{"id":"sheet-1","name":"Sheet","mimeType":"application/vnd.google-apps.spreadsheet","modifiedTime":"2026-05-29T12:05:00Z"},{"id":"slide-1","name":"Slides","mimeType":"application/vnd.google-apps.presentation","modifiedTime":"2026-05-29T12:10:00Z"}]}`))
	})
	mux.HandleFunc("/drive/v3/files/doc-1/export", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Doc body"))
	})
	mux.HandleFunc("/drive/v3/files/sheet-1/export", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("name,value\nalpha,1\nbeta,2\n"))
	})
	mux.HandleFunc("/slides/v1/presentations/slide-1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"slides":[{"pageElements":[{"shape":{"text":{"textElements":[{"textRun":{"content":"Title\n"}},{"textRun":{"content":"Bullet one\n"}}]}}}]},{"pageElements":[{"shape":{"text":{"textElements":[{"textRun":{"content":"Follow-up\n"}}]}}}]}]}`))
	})
	server.Server = httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// writeAuthorizedUserCredentials writes an authorized_user credentials JSON file for tests.
func writeAuthorizedUserCredentials(t *testing.T, tokenURI string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "authorized_user.json")
	body, err := json.Marshal(map[string]string{
		"type":          "authorized_user",
		"client_id":     "client-id",
		"client_secret": "client-secret",
		"refresh_token": "refresh-token",
		"token_uri":     tokenURI,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

// writeServiceAccountCredentials writes a service_account credentials JSON file for tests.
func writeServiceAccountCredentials(t *testing.T, tokenURI string) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("x509.MarshalPKCS8PrivateKey() error = %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})

	path := filepath.Join(t.TempDir(), "service_account.json")
	body, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"client_email": "service-account@example.com",
		"private_key":  string(pemBytes),
		"token_uri":    tokenURI,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

// findContent returns the content for the event with the given Google Drive file ID.
func findContent(ingested []events.Event, fileID string) string {
	for _, event := range ingested {
		if event.Metadata[metadataFileID] == fileID {
			return event.Content
		}
	}
	return ""
}
