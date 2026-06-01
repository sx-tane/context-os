# AI Worker Client

`internal/aiworker` is a thin, synchronous HTTP client for the local ContextOS
AI worker. It lets opt-in callers reach the worker's deterministic embedding
endpoint without making any pipeline stage depend on the network.

## Responsibility

- Call the worker's `POST /embed` endpoint and return one vector per input text.
- Satisfy the `identity.Embedder` interface so the identity stage's opt-in
  `WorkerMatcher` can request semantic embeddings.
- Stay synchronous so callers decide whether to run it in a goroutine.
- Keep the pipeline hermetic by default: no stage constructs a `Client` unless a
  caller explicitly opts in.

## Configuration

The worker base URL is resolved in this order:

1. `WithBaseURL(...)` option.
2. `WORKER_URL` environment variable.
3. `http://localhost:8081` default (the worker's default bind address).

`docker-compose.yml` sets `WORKER_URL=http://worker:8081` on the `api` service so
the containerised API can reach the worker by service name.

## Key API

```go
func New(opts ...Option) *Client
func WithBaseURL(url string) Option
func WithHTTPClient(h *http.Client) Option

func (c *Client) Embed(texts []string) ([][]float64, error)
func (c *Client) EmbedContext(ctx context.Context, texts []string) ([][]float64, error)
```

## Contract

```mermaid
flowchart LR
  caller["identity.WorkerMatcher"]
  client["aiworker.Client"]
  worker["ai-worker POST /embed"]

  caller -->|Embed texts| client -->|JSON| worker
  worker -->|vectors| client -->|[][]float64| caller
```

`Embed` returns an error when the worker responds with a non-200 status or a
vector count that does not match the input length. Empty input returns no vectors
without contacting the worker. The matching worker endpoint contract lives in
[`apps/ai-worker/README.md`](../../apps/ai-worker/README.md).

## Tests

```sh
go test ./internal/aiworker/
```

Tests use `httptest` stub servers; no real worker is required.
