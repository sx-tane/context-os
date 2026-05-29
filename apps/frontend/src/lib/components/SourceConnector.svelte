<script lang="ts">
  import { onDestroy } from "svelte";
  import type {
    DirectSourceConnectorKind,
    IngestResult,
    SourceMetadataField,
    SupportedFormat,
  } from "$lib/types";
  import { postFilesystemUpload } from "$lib/api";
  import { runConnectorIngest } from "$lib/ingestRunner";
  import ConnectorCard from "./ConnectorCard.svelte";
  import ResultPanel from "./IngestResult.svelte";

  export let connector: DirectSourceConnectorKind;
  export let title: string;
  export let description: string;
  export let examples: string[] = [];
  export let defaultUri = "";
  export let uriLabel = "URI";
  export let uriPlaceholder = "";
  export let submitLabel = "Run ingest";
  export let uploadEnabled = false;
  export let tokenLabel = "";
  export let tokenPlaceholder = "";
  export let contentLabel = "";
  export let contentPlaceholder = "";
  export let metadataFields: SourceMetadataField[] = [];
  export let supportedFormats: SupportedFormat[] = [];

  let uri = defaultUri;
  let token = "";
  let content = "";
  let uploadFiles: File[] = [];
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

  function selectUploadFiles(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    uploadFiles = Array.from(input.files ?? []);
    errorMessage = "";
    result = null;
  }

  function uploadFilePath(file: File): string {
    return file.webkitRelativePath || file.name;
  }

  $: uploadSummary =
    uploadFiles.length === 0
      ? ""
      : uploadFiles.length === 1
        ? uploadFilePath(uploadFiles[0])
        : `${uploadFiles.length} files selected`;

  async function runUploadIngest() {
    if (uploadFiles.length === 0) {
      errorMessage = "Choose at least one file or folder first.";
      return;
    }

    ingestController?.abort();
    ingestController = new AbortController();
    const runID = ++ingestRunID;
    loading = true;
    errorMessage = "";
    result = null;
    elapsed = 0;

    try {
      const formData = new FormData();
      for (const file of uploadFiles) {
        formData.append("files", file, file.name);
        formData.append("paths", uploadFilePath(file));
      }
      const res = await postFilesystemUpload(formData, {
        signal: ingestController.signal,
      });
      if (runID !== ingestRunID) return;
      if (!res.ok) {
        errorMessage =
          res.body?.message ?? `Request failed with status ${res.status}`;
        return;
      }
      result = res.body;
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      if (runID !== ingestRunID) return;
      errorMessage = err instanceof Error ? err.message : String(err);
    } finally {
      if (runID === ingestRunID) loading = false;
    }
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
  {#if uploadEnabled}
    <div class="connector-upload">
      <div class="connector-upload-actions">
        <label class="connector-button connector-file-button">
          Choose files
          <input
            class="connector-file-input"
            type="file"
            multiple
            on:change={selectUploadFiles}
          />
        </label>
        <label
          class="connector-button connector-button-secondary connector-file-button"
        >
          Choose folder
          <input
            class="connector-file-input"
            type="file"
            multiple
            webkitdirectory
            on:change={selectUploadFiles}
          />
        </label>
      </div>

      {#if uploadSummary}
        <div class="connector-upload-summary">{uploadSummary}</div>
      {/if}

      <button
        class="connector-button"
        type="button"
        on:click={runUploadIngest}
        disabled={loading || uploadFiles.length === 0}
      >
        {loading ? `Ingesting... (${elapsed}s)` : "Upload and ingest"}
      </button>
    </div>

    <details class="connector-path-fallback">
      <summary>Use server path</summary>
      <label class="connector-field connector-field-offset">
        <span>{uriLabel}</span>
        <input
          class="connector-input"
          type="text"
          bind:value={uri}
          placeholder={uriPlaceholder}
        />
      </label>
      <button
        class="connector-button"
        type="button"
        on:click={runIngest}
        disabled={loading || !uri.trim()}
      >
        {loading ? `Ingesting... (${elapsed}s)` : submitLabel}
      </button>
    </details>
  {:else}
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

  {#if !uploadEnabled}
    <button
      class="connector-button"
      type="button"
      on:click={runIngest}
      disabled={loading || (!uri.trim() && !content.trim())}
    >
      {loading ? `Ingesting... (${elapsed}s)` : submitLabel}
    </button>
  {/if}

  {#if errorMessage}
    <div class="connector-error">{errorMessage}</div>
  {/if}

  {#if result}
    <ResultPanel {result} provider="token" />
  {/if}
</ConnectorCard>
