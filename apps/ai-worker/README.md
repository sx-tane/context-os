# AI Worker App

Python worker surface for local-first AI-assisted classification, extraction, and reasoning tasks.

Production responsibility:

- run optional AI-assisted work without replacing deterministic evidence;
- preserve prompts, source references, outputs, and trace metadata;
- support replay and cancellation;
- return assistive evidence that reasoning can inspect and audit.

## Run Commands

```bash
cd apps/ai-worker
uv sync
uv run python health.py
```

## Endpoints

| Method | Path      | Purpose                                                  |
| ------ | --------- | -------------------------------------------------------- |
| GET    | `/health` | Liveness check. Returns `{"status":"ok","service":...}`. |
| POST   | `/embed`  | Deterministic text embedding. See contract below.        |

### POST /embed

Request body:

```json
{ "texts": ["refund_status", "refundStatus"] }
```

Response body:

```json
{ "model": "contextos-hashing-v1", "dim": 256, "vectors": [[...], [...]] }
```

Embeddings are produced by [`embed.py`](embed.py): character trigrams are hashed
into a fixed `dim`-length vector and L2-normalised, so the same text always
yields the same vector with no external dependencies. The `model` field is an
extension point for swapping in a real transformer without changing the
request/response contract. Bodies are capped at 1 MiB.

The Go side consumes this endpoint through
[`internal/aiworker`](../../internal/aiworker/README.md), which the identity
stage's opt-in `WorkerMatcher` uses for semantic candidate detection.

## Tests

```bash
cd apps/ai-worker
python -m unittest test_embed
```

## Maintenance Checklist

- Keep worker behavior assistive and explainable rather than authoritative.
- Document new worker entrypoints or runtime dependencies here.
- Preserve local-first execution assumptions when adding integrations.
