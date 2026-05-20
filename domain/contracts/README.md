# Domain Contracts

Package `domain/contracts` defines the source connector boundary. Anything that can provide external context to ContextOS should implement this package, then be registered with the ingestion pipeline.

## Responsibility

- Describe source connector capabilities.
- Carry source input through `SourceRequest`.
- Define the `MCPSourceConnector` interface used by ingestion.

## Key Types

```go
type Capability string

const (
    CapabilityRepository  Capability = "repository"
    CapabilityMessages    Capability = "messages"
    CapabilityIssues      Capability = "issues"
    CapabilityAPISpec     Capability = "api_spec"
    CapabilitySpreadsheet Capability = "spreadsheet"
    CapabilityFiles       Capability = "files"
)
```

Capabilities are descriptive routing hints. They should reflect what a connector can ingest, not the current request content.

```go
type SourceRequest struct {
    URI      string            `json:"uri"`
    Content  string            `json:"content"`
    Metadata map[string]string `json:"metadata"`
}
```

`SourceRequest` is the universal source input envelope. `URI` identifies an external resource when available. `Content` carries inline payloads. `Metadata` carries source-specific context without changing the contract.

```go
type MCPSourceConnector interface {
    Name() string
    Capabilities() []Capability
    Ingest(context.Context, SourceRequest) ([]events.Event, error)
}
```

`MCPSourceConnector` converts source input into domain events. Implementations should be idempotent when the same `SourceRequest` is replayed.

## Inputs And Outputs

```mermaid
flowchart LR
  request[SourceRequest]
  connector[MCPSourceConnector]
  event[[]events.Event]

  request --> connector --> event
```

## Implementation Notes

- Connector implementations live under [internal/source](../../internal/source/README.md).
- `Ingest` should respect context cancellation before doing work.
- Metadata should preserve provenance such as connector name, external ID, and source timestamps where available.
- For replay safety, prefer stable source identifiers over generated identifiers whenever the external system provides them.
