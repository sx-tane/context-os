# GitHub Source

Source connector for GitHub repositories, issues, and pull requests.

## What It Does

- Exposes the public connector name `github` with `repository` capability.
- Accepts `repo://`, `github://`, `https://github.com`, and GitHub API URLs.
- Enriches metadata with owner, repo, artifact number, object type, object ID, and stable `source_id`.
- Fetches repository, issue, or pull request JSON when content is not provided.
- Uses `GITHUB_TOKEN` or request metadata token for authenticated API reads.

## Important Files

| File             | Role                                                                      |
| ---------------- | ------------------------------------------------------------------------- |
| `github.go`      | URI parsing, API fetch, metadata enrichment, structured connector errors. |
| `github_test.go` | Provenance, URI parsing, auth, API error, and replay behavior tests.      |

## Replay Notes

Issue and pull request reads use upstream update timestamps as cursors when available. Stable event identity comes from URI, content, cursor, and source metadata.
