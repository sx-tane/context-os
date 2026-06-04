<script lang="ts">
  import type { EvidenceBasketItem } from "$lib/workflow/types";
  import type {
    AnalysisPreviewModel,
    AnalysisPreviewRow,
    SourceHealthRow,
  } from "$lib/workflow/viewModel";

  export let preview: AnalysisPreviewModel;
  export let sourceHealth: SourceHealthRow[] = [];
  export let basketItems: EvidenceBasketItem[] = [];
  export let onRemoveBasketItem: (id: string) => void | Promise<void> = () => {};
  export let onExportMarkdown: () => void | Promise<void> = () => {};

  function rowTitle(row: AnalysisPreviewRow | SourceHealthRow) {
    return `${row.connector}:${row.uri}`;
  }
</script>

<section class="analysis-preview" aria-label="Analysis preview">
  <div class="preview-head">
    <div>
      <strong>Analysis Preview</strong>
      <span>{preview.summary}</span>
    </div>
    <button type="button" on:click={onExportMarkdown}>Export Markdown</button>
  </div>

  {#if basketItems.length}
    <section class="preview-section">
      <div class="section-head">
        <strong>Evidence Basket</strong>
        <span>{basketItems.length} pinned</span>
      </div>
      <div class="preview-rows">
        {#each basketItems as item (item.id)}
          <div class={`preview-row source-${item.connector}`}>
            <div>
              <strong class="source-title">{item.label}</strong>
              <small>{item.connector}:{item.uri}</small>
            </div>
            <button type="button" on:click={() => onRemoveBasketItem(item.id)}>
              Remove
            </button>
          </div>
        {/each}
      </div>
    </section>
  {/if}

  <div class="preview-columns">
    <section class="preview-section">
      <div class="section-head">
        <strong>Included</strong>
        <span>{preview.included.length}</span>
      </div>
      <div class="preview-rows">
        {#if preview.included.length}
          {#each preview.included as row (row.id)}
            <div class={`preview-row source-${row.connector}`}>
              <div>
                <strong class="source-title">{rowTitle(row)}</strong>
                <small>{row.origin}</small>
              </div>
            </div>
          {/each}
        {:else}
          <p>No concrete source will be analyzed yet.</p>
        {/if}
      </div>
    </section>

    <section class="preview-section">
      <div class="section-head">
        <strong>{preview.hasBasketSelection ? "Available, Not Selected" : "Available"}</strong>
        <span>{preview.available.length}</span>
      </div>
      <div class="preview-rows">
        {#if preview.available.length}
          {#each preview.available as row (row.id)}
            <div class={`preview-row source-${row.connector}`}>
              <div>
                <strong class="source-title">{rowTitle(row)}</strong>
                <small>{row.origin}</small>
              </div>
            </div>
          {/each}
        {:else}
          <p>Ask chat about a concrete ticket, channel, repo, or file to create analysis-ready evidence.</p>
        {/if}
      </div>
    </section>

    <section class="preview-section">
      <div class="section-head">
        <strong>Chat-Only</strong>
        <span>{preview.skipped.length}</span>
      </div>
      <div class="preview-rows">
        {#if preview.skipped.length}
          {#each preview.skipped as row (row.id)}
            <div class={`preview-row source-${row.connector}`}>
              <div>
                <strong class="source-title">{rowTitle(row)}</strong>
                <small>Broad connector scope</small>
              </div>
            </div>
          {/each}
        {:else}
          <p>No broad-only connector scopes are being skipped.</p>
        {/if}
      </div>
    </section>
  </div>

  <section class="preview-section health-section">
    <div class="section-head">
      <strong>Source Health</strong>
      <span>{sourceHealth.length}</span>
    </div>
    <div class="health-rows">
      {#if sourceHealth.length}
        {#each sourceHealth as row (row.id)}
          <div class={`health-row source-${row.connector}`}>
            <div>
              <strong class="source-title">{row.label}</strong>
              <small>{row.detail}</small>
            </div>
            <span class:attention={row.status === "needs-attention"}>
              {row.status}
            </span>
          </div>
        {/each}
      {:else}
        <p>No source health rows yet.</p>
      {/if}
    </div>
  </section>
</section>

<style>
  button {
    font: inherit;
    cursor: pointer;
  }

  .analysis-preview {
    border-bottom: 1px solid #d7d2c8;
    padding: 12px 0 14px;
  }

  .preview-head,
  .section-head,
  .preview-row,
  .health-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    min-width: 0;
  }

  .preview-head {
    border-bottom: 1px solid #d7d2c8;
    padding-bottom: 10px;
  }

  .preview-head strong,
  .preview-head span,
  .section-head strong,
  .section-head span,
  .preview-row strong,
  .preview-row small,
  .health-row strong,
  .health-row small {
    display: block;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .preview-head span,
  .section-head span,
  .preview-row small,
  .health-row small {
    color: #8a8678;
    font-size: 11px;
    text-transform: uppercase;
  }

  .preview-head button,
  .preview-row button {
    flex: 0 0 auto;
    border: 0;
    border-bottom: 1px solid #bdb7a8;
    border-radius: 0;
    background: transparent;
    color: #1c1b18;
    font-size: 11px;
    font-weight: 700;
    padding: 4px 0;
  }

  .preview-head button:hover,
  .preview-row button:hover {
    border-bottom-color: #1c1b18;
  }

  .preview-columns {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 16px;
  }

  .preview-section {
    min-width: 0;
    border-bottom: 1px solid rgba(215, 210, 200, 0.72);
    padding: 12px 0;
  }

  .health-section {
    border-bottom: 0;
    padding-bottom: 0;
  }

  .section-head {
    align-items: baseline;
  }

  .preview-rows,
  .health-rows {
    display: grid;
    gap: 0;
    margin-top: 8px;
  }

  .preview-row,
  .health-row {
    border-top: 1px solid rgba(228, 222, 210, 0.88);
    padding: 8px 0;
  }

  .source-title {
    color: #1c1b18;
  }

  .source-jira .source-title {
    color: #0c66e4;
  }

  .source-github .source-title {
    color: #24292f;
  }

  .source-slack .source-title {
    color: #4a154b;
  }

  .source-googledrive .source-title {
    color: #1a73e8;
  }

  .source-notion .source-title {
    color: #1c1b18;
  }

  .source-sharepoint .source-title {
    color: #036c70;
  }

  .source-filesystem .source-title {
    color: #8a6a20;
  }

  .health-row > span {
    flex: 0 0 auto;
    color: #2d6a4f;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .health-row > span.attention {
    color: #b5523a;
  }

  p {
    margin: 8px 0 0;
    color: #5f5b50;
    font-size: 12px;
    line-height: 1.45;
  }

  @media (max-width: 760px) {
    .preview-columns {
      grid-template-columns: 1fr;
      gap: 0;
    }
  }
</style>
