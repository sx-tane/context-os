# Slack Source

Source connector for Slack channels and messages.

## What It Does

- Exposes the public connector name `slack` with `messages` capability.
- Accepts `slack://CHANNEL_ID` and `slack://CHANNEL_ID/TIMESTAMP` URIs.
- Enriches metadata with channel ID, message timestamp, object type, object ID, and stable `source_id`.
- Fetches channel or message JSON when content is not provided.
- Uses saved OAuth, `SLACK_BOT_TOKEN`, or request metadata token for authenticated reads.

## Important Files

| File            | Role                                                                  |
| --------------- | --------------------------------------------------------------------- |
| `slack.go`      | Slack API fetch, URI parsing, metadata enrichment, structured errors. |
| `slack_test.go` | Slack provenance, API, auth, and replay behavior tests.               |

## Replay Notes

Message reads use Slack timestamps as cursors when available. Channel reads keep stable source IDs based on channel identity.
