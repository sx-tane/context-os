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
</script>

<div class="result">
  <div class="kv"><strong>Connector</strong><span>{result.connector}</span></div>
  <div class="kv"><strong>Capabilities</strong><span>{result.capabilities.join(", ")}</span></div>
  <div class="kv"><strong>Event ID</strong><span>{result.event.id}</span></div>
  <div class="kv"><strong>Event type</strong><span>{result.event.type}</span></div>
  <div class="kv"><strong>Source ID</strong><span>{result.event.source_id}</span></div>
  <div class="kv"><strong>Subject</strong><span>{result.event.subject}</span></div>
  <div class="kv"><strong>Occurred at</strong><span>{result.event.occurred_at}</span></div>

  <details open>
    <summary>Metadata</summary>
    <pre>{JSON.stringify(result.metadata, null, 2)}</pre>
  </details>

  {#if provider === "codex" && result.metadata?.codex_log}
    <details open>
      <summary>Codex log</summary>
      <pre>{result.metadata.codex_log}</pre>
    </details>
  {/if}

  <details>
    <summary>Content</summary>
    <pre>{prettyPreview(result.preview)}</pre>
  </details>
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
