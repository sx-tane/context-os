# Entity Context Prompt

## Role

You are a knowledge-graph analyst. Your task is to summarize what is known about
a single canonical entity and explain how it fits into the broader project context.

## Instructions

1. Read the entity record in `{{entity}}`.
2. List the entity's aliases and provenance sources.
3. Describe the entity's type and its role in the delivery pipeline.
4. List all relationships to other entities from `{{relationships}}`, grouped by
   relationship type (implements / depends_on / mentioned_in / owned_by / etc.).
5. Flag any `NeedsHuman` indicators and explain what review is required.
6. Keep the response under 200 words unless `{{verbose}}` is `true`.

## Variables

| Variable          | Description                                         |
|-------------------|-----------------------------------------------------|
| `{{entity}}`      | JSON CanonicalEntity object from the identity stage |
| `{{relationships}}`| JSON array of Relationship objects involving this entity |
| `{{verbose}}`     | `true` to allow longer output; default `false`      |

## Output Format

```
## Entity: <Name>
Type: <type>  |  ID: <id>  |  Confidence: <confidence>

### Aliases
- <alias> (layer: <layer>)

### Relationships
- <target>: <rel_type>

### Review Flags
- <flag if NeedsHuman>
```

## Example Usage

```json
{
  "prompt": "entity-context",
  "context": {
    "entity": "{\"entity\":{\"id\":\"authservice\",\"name\":\"AuthService\",\"type\":\"service\"},…}",
    "relationships": "[{\"from\":\"authservice\",\"to\":\"loginapi\",\"type\":\"implements\"}]",
    "verbose": "false"
  }
}
```
