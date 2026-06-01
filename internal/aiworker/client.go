// Package aiworker is a thin HTTP client for the local ContextOS AI worker. It
// exposes the worker's deterministic embedding endpoint to Go callers without
// pulling any pipeline stage into a network dependency: only callers that opt in
// (such as the identity stage's WorkerMatcher) construct and pass a Client. The
// client is synchronous so callers decide whether to run it in a goroutine.
package aiworker

import (
	"bytes"         // builds the JSON request body
	"context"       // cancellation and deadline propagation
	"encoding/json" // encodes requests and decodes responses
	"fmt"           // wraps errors with context
	"net/http"      // performs the HTTP call to the worker
	"os"            // reads the WORKER_URL environment override
	"strings"       // trims a trailing slash from the base URL
)

// defaultBaseURL is the worker address used when WORKER_URL is unset. It matches
// the worker's default bind port for local-first single-machine runs.
const defaultBaseURL = "http://localhost:8081"

// Client calls the ContextOS AI worker. The zero value is not usable; construct
// one with New.
type Client struct {
	baseURL string       // worker base URL without a trailing slash
	http    *http.Client // underlying HTTP client, injectable for tests
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the HTTP client used for requests, mainly for tests.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithBaseURL overrides the worker base URL, taking precedence over WORKER_URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// New builds a Client. The base URL is taken from the WithBaseURL option, then
// the WORKER_URL environment variable, then the local default.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL: envOrDefault("WORKER_URL", defaultBaseURL),
		http:    http.DefaultClient,
	}
	for _, opt := range opts {
		opt(client)
	}
	client.baseURL = strings.TrimRight(client.baseURL, "/")
	return client
}

// embedRequest is the JSON body sent to POST /embed.
type embedRequest struct {
	Texts []string `json:"texts"`
}

// embedResponse is the JSON body returned by POST /embed.
type embedResponse struct {
	Model   string      `json:"model"`
	Dim     int         `json:"dim"`
	Vectors [][]float64 `json:"vectors"`
}

// Embed returns one embedding vector per input text, preserving order. It
// satisfies the identity.Embedder interface. An empty input returns no vectors
// without contacting the worker.
func (c *Client) Embed(texts []string) ([][]float64, error) {
	return c.EmbedContext(context.Background(), texts)
}

// EmbedContext is Embed with caller-supplied cancellation and deadline control.
func (c *Client) EmbedContext(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(embedRequest{Texts: texts})
	if err != nil {
		return nil, fmt.Errorf("aiworker: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("aiworker: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aiworker: call worker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aiworker: worker returned status %d", resp.StatusCode)
	}

	var decoded embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("aiworker: decode response: %w", err)
	}
	if len(decoded.Vectors) != len(texts) {
		return nil, fmt.Errorf("aiworker: worker returned %d vectors, want %d", len(decoded.Vectors), len(texts))
	}
	return decoded.Vectors, nil
}

// envOrDefault returns the environment value for key, or fallback when unset or empty.
func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
