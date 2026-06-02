<script lang="ts">
  import type { IngestResult } from "$lib/types";

  export let result: IngestResult;
  export let provider: string;

  function prettyPreview(raw: string): string {
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw;
    }
  }

  $: events = result.events?.length ? result.events : result.event ? [result.event] : [];
  $: previews = result.previews?.length ? result.previews : result.preview ? [result.preview] : [];
  $: metadataItems = result.metadata_items?.length
    ? result.metadata_items
    : [result.metadata];
  $: eventCount = result.event_count ?? events.length;
</script>

<div class="result">
  <div class="kv">
    <strong>Connector</strong><span>{result.connector}</span>
  </div>
  <div class="kv">
    <strong>Persistence</strong><span>{result.persistence_mode ?? "preview_debug"}</span>
  </div>
  {#if result.workspace_id}
    <div class="kv">
      <strong>Workspace DB</strong><span>{result.workspace_id}</span>
    </div>
  {/if}
  <div class="kv">
    <strong>Capabilities</strong><span>{result.capabilities?.join(", ") ?? ""}</span>
  </div>
  {#if result.persisted_event_count !== undefined}
    <div class="kv"><strong>Persisted events</strong><span>{result.persisted_event_count}</span></div>
  {/if}
  {#if result.relationship_count !== undefined}
    <div class="kv"><strong>Relationships</strong><span>{result.relationship_count}</span></div>
  {/if}
  {#if eventCount > 1}
    <div class="kv"><strong>Event count</strong><span>{eventCount}</span></div>
  {/if}
  <div class="kv"><strong>Event ID</strong><span>{result.event?.id ?? ""}</span></div>
  <div class="kv">
    <strong>Event type</strong><span>{result.event?.type ?? ""}</span>
  </div>
  <div class="kv">
    <strong>Source ID</strong><span>{result.event?.source_id ?? ""}</span>
  </div>
  <div class="kv">
    <strong>Subject</strong><span>{result.event?.subject ?? ""}</span>
  </div>
  <div class="kv">
    <strong>Occurred at</strong><span>{result.event?.occurred_at ?? ""}</span>
  </div>

  {#if eventCount > 1}
    <details open>
      <summary>Events</summary>
      <div class="event-table-wrap">
        <table class="event-table">
          <thead>
            <tr>
              <th>#</th>
              <th>Subject</th>
              <th>Format</th>
              <th>Source ID</th>
            </tr>
          </thead>
          <tbody>
            {#each events as event, index}
              <tr>
                <td>{index + 1}</td>
                <td>{event?.subject ?? ""}</td>
                <td>{metadataItems[index]?.filesystem_format ?? ""}</td>
                <td>{event?.source_id ?? ""}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </details>
  {/if}

  <details open>
    <summary>{eventCount > 1 ? "First metadata" : "Metadata"}</summary>
    <pre>{JSON.stringify(result.metadata, null, 2)}</pre>
  </details>

  {#if provider === "codex" && result.metadata?.codex_log}
    <details open>
      <summary>Codex log</summary>
      <pre>{result.metadata.codex_log}</pre>
    </details>
  {/if}

  <details>
    <summary>{eventCount > 1 ? "First content" : "Content"}</summary>
    <pre>{prettyPreview(result.preview ?? "")}</pre>
  </details>

  {#if eventCount > 1}
    <details>
      <summary>Previews</summary>
      {#each previews as item, index}
        <div class="preview-heading">
          {events[index]?.subject ?? `Event ${index + 1}`}
        </div>
        <pre>{prettyPreview(item ?? "")}</pre>
      {/each}
    </details>
  {/if}
</div>

<style>
  .result {
    margin-top: 1rem;
    border-top: 1px solid #e5e7eb;
    padding-top: 1rem;
  }

  .kv {
    display: flex;
    gap: 0.75rem;
    padding: 0.25rem 0;
    font-size: 0.85rem;
    word-break: break-all;
  }

  .kv strong {
    color: #374151;
    min-width: 8rem;
  }

  details {
    margin-top: 0.75rem;
    font-size: 0.85rem;
  }

  .event-table-wrap {
    overflow-x: auto;
  }

  .event-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.8rem;
  }

  .event-table th,
  .event-table td {
    border: 1px solid #e5e7eb;
    padding: 0.4rem 0.5rem;
    text-align: left;
    vertical-align: top;
    word-break: break-all;
  }

  .event-table th {
    background: #f9fafb;
    color: #374151;
  }

  .preview-heading {
    color: #374151;
    font-size: 0.8rem;
    font-weight: 600;
    margin-top: 0.75rem;
    word-break: break-all;
  }

  pre {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    padding: 0.75rem;
    border-radius: 6px;
    max-height: 320px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }
</style>
