// Package aiworker_test verifies the AI worker HTTP client against a stub server.
package aiworker_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"context-os/internal/aiworker"
)

// TestEmbedReturnsVectorsInOrder verifies Embed posts the texts and returns one
// vector per input in order.
func TestEmbedReturnsVectorsInOrder(t *testing.T) {
	var gotTexts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/embed" {
			t.Errorf("request = %s %s, want POST /embed", r.Method, r.URL.Path)
		}
		var body struct {
			Texts []string `json:"texts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request error = %v", err)
		}
		gotTexts = body.Texts
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "test",
			"dim":     2,
			"vectors": [][]float64{{1, 0}, {0, 1}},
		})
	}))
	defer server.Close()

	client := aiworker.New(aiworker.WithBaseURL(server.URL))
	vectors, err := client.Embed([]string{"refund_status", "invoice_total"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("vectors length = %d, want 2", len(vectors))
	}
	if len(gotTexts) != 2 || gotTexts[0] != "refund_status" {
		t.Errorf("posted texts = %v, want [refund_status invoice_total]", gotTexts)
	}
	if vectors[0][0] != 1 || vectors[1][1] != 1 {
		t.Errorf("vectors = %v, want [[1 0] [0 1]]", vectors)
	}
}

// TestEmbedEmptyInputSkipsCall verifies an empty input returns no vectors without
// contacting the worker.
func TestEmbedEmptyInputSkipsCall(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client := aiworker.New(aiworker.WithBaseURL(server.URL))
	vectors, err := client.Embed(nil)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if vectors != nil {
		t.Errorf("vectors = %v, want nil", vectors)
	}
	if called {
		t.Errorf("worker called = true, want false for empty input")
	}
}

// TestEmbedRejectsCountMismatch verifies a vector-count mismatch is an error.
func TestEmbedRejectsCountMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "test",
			"dim":     2,
			"vectors": [][]float64{{1, 0}},
		})
	}))
	defer server.Close()

	client := aiworker.New(aiworker.WithBaseURL(server.URL))
	if _, err := client.Embed([]string{"a", "b"}); err == nil {
		t.Fatalf("Embed() error = nil, want count mismatch error")
	}
}

// TestEmbedRejectsNon200 verifies a non-200 worker status is an error.
func TestEmbedRejectsNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := aiworker.New(aiworker.WithBaseURL(server.URL))
	if _, err := client.EmbedContext(context.Background(), []string{"a"}); err == nil {
		t.Fatalf("Embed() error = nil, want status error")
	}
}
