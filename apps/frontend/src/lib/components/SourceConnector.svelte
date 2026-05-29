<script lang="ts">
  import { onDestroy } from "svelte";
  import type {
    ConnectorKind,
    IngestResult,
    SourceMetadataField,
    SupportedFormat,
  } from "$lib/types";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import ResultPanel from "./IngestResult.svelte";

  export let connector: ConnectorKind;
  export let title: string;
  export let description: string;
  export let examples: string[] = [];
  export let defaultUri = "";
  export let uriLabel = "URI";
  export let uriPlaceholder = "";
  export let submitLabel = "Run ingest";
  export let tokenLabel = "";
  export let tokenPlaceholder = "";
  export let contentLabel = "";
  export let contentPlaceholder = "";
  export let metadataFields: SourceMetadataField[] = [];
  export let supportedFormats: SupportedFormat[] = [];

  let uri = defaultUri;
  let token = "";
  let content = "";
  let metadataValues: Record<string, string> = {};
  let loading = false;
  let errorMessage = "";
  let result: IngestResult | null = null;
  let elapsed = 0;
  let ingestController: AbortController | null = null;
  let ingestRunID = 0;

  $: metadataValues = withDefaultMetadata(metadataFields, metadataValues);

  onDestroy(() => ingestController?.abort());

  function withDefaultMetadata(
    fields: SourceMetadataField[],
    current: Record<string, string>,
  ): Record<string, string> {
    const next = { ...current };
    for (const field of fields) {
      if (next[field.key] === undefined) {
        next[field.key] = field.defaultValue ?? "";
      }
    }
    return next;
  }

  function updateMetadata(key: string, event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    metadataValues = { ...metadataValues, [key]: input.value };
  }

  async function runIngest() {
    ingestController?.abort();
    ingestController = new AbortController();
    const runID = ++ingestRunID;
    await runConnectorIngest({
      connector,
      uri,
      token,
      content,
      provider: "token",
      metadata: metadataValues,
      signal: ingestController.signal,
      isCurrent: () => runID === ingestRunID,
      setLoading: (value) => (loading = value),
      setError: (message) => (errorMessage = message),
      setResult: (value) => (result = value),
      setLiveLog: () => undefined,
      setElapsed: (value) =>
        (elapsed = typeof value === "function" ? value(elapsed) : value),
    });
  }
</script>

<ConnectorCard {title} {description} {examples}>
  <label class="connector-field">
    <span>{uriLabel}</span>
    <input
      class="connector-input"
      type="text"
      bind:value={uri}
      placeholder={uriPlaceholder}
    />
  </label>

  {#if tokenLabel}
    <label class="connector-field">
      <span>{tokenLabel}</span>
      <input
        class="connector-input"
        type="password"
        bind:value={token}
        placeholder={tokenPlaceholder}
      />
    </label>
  {/if}

  {#each metadataFields as field}
    <label class="connector-field">
      <span>{field.label}</span>
      <input
        class="connector-input"
        type={field.type ?? "text"}
        value={metadataValues[field.key] ?? ""}
        placeholder={field.placeholder ?? ""}
        on:input={(event) => updateMetadata(field.key, event)}
      />
    </label>
  {/each}

  {#if contentLabel}
    <label class="connector-field">
      <span>{contentLabel}</span>
      <textarea
        class="connector-textarea"
        bind:value={content}
        placeholder={contentPlaceholder}
      ></textarea>
    </label>
  {/if}

  {#if supportedFormats.length > 0}
    <details class="connector-formats">
      <summary>Supported formats</summary>
      <div class="connector-table-wrap">
        <table class="connector-format-table">
          <thead>
            <tr>
              <th>Format</th>
              <th>Extensions</th>
              <th>Extraction</th>
            </tr>
          </thead>
          <tbody>
            {#each supportedFormats as row}
              <tr>
                <td>{row.format}</td>
                <td
                  ><code class="connector-card-code">{row.extensions}</code></td
                >
                <td>{row.extraction}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </details>
  {/if}

  <button
    class="connector-button"
    type="button"
    on:click={runIngest}
    disabled={loading || (!uri.trim() && !content.trim())}
  >
    {loading ? `Ingesting... (${elapsed}s)` : submitLabel}
  </button>

  {#if errorMessage}
    <div class="connector-error">{errorMessage}</div>
  {/if}

  {#if result}
    <ResultPanel {result} provider="token" />
  {/if}
</ConnectorCard>
