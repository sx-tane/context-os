<script lang="ts">
  import type { Artifact } from "$lib/types";
  import type { EvidenceBasketItem } from "$lib/workflow/types";
  import ConfirmModal from "$lib/components/ui/ConfirmModal.svelte";
  import SafeMarkdownBlock from "$lib/components/ui/SafeMarkdownBlock.svelte";
  import {
    activityFilterLabel,
    activityEventSummary,
    artifactDetailRows,
    artifactLink,
    artifactOrigin,
    artifactProvider,
    formatTime,
    filterArtifactsByTime,
    groupArtifactsBySource,
    markdownBulletList,
    normalizeActivityTimeFilter,
    previewMarkdownText,
    previewText,
    type ActivityTimeFilter,
  } from "$lib/findings/viewModel";
  import {
    activityEvidenceType,
    askChatPromptForEvidence,
    basketItemFromArtifact,
    filterActivityArtifacts,
    type ActivityFilterState,
  } from "$lib/workflow/viewModel";

  export let recentArtifacts: Artifact[] = [];
  export let basketItems: EvidenceBasketItem[] = [];
  export let onCleanupNoisyLiveEvidence: () => Promise<string> = async () => "";
  export let onAskEvidence: (prompt: string) => void | Promise<void> = () => {};
  export let onPinEvidence: (item: EvidenceBasketItem) => void | Promise<void> = () => {};

  const filterStorageKey = "contextos_activity_time_filter";
  const filters: ActivityTimeFilter[] = ["24h", "7d", "30d", "all"];

  let timeFilter: ActivityTimeFilter = "7d";
  let selectedArtifactID = "";
  let cleanupConfirmOpen = false;
  let cleanupRunning = false;
  let cleanupMessage = "";
  let activityFilters: ActivityFilterState = {};

  $: timeFilteredArtifacts = filterArtifactsByTime(recentArtifacts, timeFilter);
  $: visibleArtifacts = filterActivityArtifacts(timeFilteredArtifacts, activityFilters);
  $: sourceGroups = groupArtifactsBySource(visibleArtifacts);
  $: connectorOptions = uniqueValues(
    recentArtifacts.map((artifact) => artifact.connector).filter(Boolean),
  );
  $: evidenceTypeOptions = uniqueValues(
    recentArtifacts.map((artifact) => activityEvidenceType(artifact)).filter(Boolean),
  );

  function loadSavedFilter() {
    if (typeof localStorage === "undefined") return;
    timeFilter = normalizeActivityTimeFilter(
      localStorage.getItem(filterStorageKey),
    );
  }

  function changeTimeFilter(value: string) {
    timeFilter = normalizeActivityTimeFilter(value);
    selectedArtifactID = "";
    if (typeof localStorage === "undefined") return;
    localStorage.setItem(filterStorageKey, timeFilter);
  }

  function updateActivityFilter(key: keyof ActivityFilterState, value: string) {
    activityFilters = {
      ...activityFilters,
      [key]: value,
    };
    selectedArtifactID = "";
  }

  function clearActivityFilters() {
    activityFilters = {};
    selectedArtifactID = "";
  }

  function toggleArtifact(artifact: Artifact) {
    selectedArtifactID = selectedArtifactID === artifact.id ? "" : artifact.id;
  }

  function connectorClass(connector?: string) {
    const normalized = (connector ?? "").toLowerCase().replace(/[^a-z0-9]/g, "");
    if (normalized === "jira") return "source-jira";
    if (normalized === "github") return "source-github";
    if (normalized === "slack") return "source-slack";
    if (normalized === "googledrive" || normalized === "google") return "source-googledrive";
    if (normalized === "notion") return "source-notion";
    if (normalized === "sharepoint" || normalized === "onedrive") return "source-sharepoint";
    if (normalized === "filesystem") return "source-filesystem";
    return "source-default";
  }

  function artifactBasketItem(artifact: Artifact) {
    return basketItemFromArtifact(artifact);
  }

  function basketHas(item: EvidenceBasketItem | null) {
    return Boolean(item && basketItems.some((existing) => existing.id === item.id));
  }

  function uniqueValues(values: string[]) {
    return [...new Set(values.map((value) => value.trim()).filter(Boolean))]
      .sort((left, right) => left.localeCompare(right));
  }

  async function confirmCleanup() {
    cleanupRunning = true;
    cleanupMessage = "";
    try {
      cleanupMessage = await onCleanupNoisyLiveEvidence();
      cleanupConfirmOpen = false;
      selectedArtifactID = "";
    } catch (error) {
      cleanupMessage = error instanceof Error ? error.message : String(error);
    } finally {
      cleanupRunning = false;
    }
  }

  loadSavedFilter();
</script>

<div class="activity-view">
  <div class="activity-filters" aria-label="Activity filters">
    <div class="filter-grid">
      <label>
        <span>Window</span>
        <select
          aria-label="Filter activity by time"
          value={timeFilter}
          on:change={(event) =>
            changeTimeFilter((event.currentTarget as HTMLSelectElement).value)}
        >
          {#each filters as filter}
            <option value={filter}>{activityFilterLabel(filter)}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Connector</span>
        <select
          aria-label="Filter activity by connector"
          value={activityFilters.connector ?? ""}
          on:change={(event) =>
            updateActivityFilter("connector", (event.currentTarget as HTMLSelectElement).value)}
        >
          <option value="">All connectors</option>
          {#each connectorOptions as connector}
            <option value={connector}>{connector}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Evidence Type</span>
        <select
          aria-label="Filter activity by evidence type"
          value={activityFilters.evidenceType ?? ""}
          on:change={(event) =>
            updateActivityFilter("evidenceType", (event.currentTarget as HTMLSelectElement).value)}
        >
          <option value="">All types</option>
          {#each evidenceTypeOptions as evidenceType}
            <option value={evidenceType}>{evidenceType}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Source URI</span>
        <input
          value={activityFilters.sourceURI ?? ""}
          placeholder="Filter source URI"
          on:input={(event) =>
            updateActivityFilter("sourceURI", (event.currentTarget as HTMLInputElement).value)}
        />
      </label>
      <label>
        <span>Keyword</span>
        <input
          value={activityFilters.keyword ?? ""}
          placeholder="Search activity"
          on:input={(event) =>
            updateActivityFilter("keyword", (event.currentTarget as HTMLInputElement).value)}
        />
      </label>
    </div>
    <div class="filter-actions">
      <button
        type="button"
        class="cleanup-action"
        aria-label="Clean noisy live evidence"
        title="Clean noisy live evidence"
        on:click={() => {
          cleanupConfirmOpen = true;
          cleanupMessage = "";
        }}
      >
        Clean Noise
      </button>
      <button type="button" on:click={clearActivityFilters}>Clear</button>
    </div>
  </div>

  {#if cleanupConfirmOpen}
    <ConfirmModal
      eyebrow="CLEAN ACTIVITY"
      title="Clean noisy live evidence?"
      description="This removes old live chat Activity rows created from duplicate full answers, URL path fragments, or generic terms. It does not run automatically."
      confirmLabel="Clean"
      busyLabel="Cleaning"
      busy={cleanupRunning}
      on:cancel={() => {
        if (!cleanupRunning) cleanupConfirmOpen = false;
      }}
      on:confirm={confirmCleanup}
    />
  {/if}

  {#if cleanupMessage}
    <p class="cleanup-message">{cleanupMessage}</p>
  {/if}

  {#if sourceGroups.length}
    {#each sourceGroups as group (group.key)}
      <section
        class={`source-group ${connectorClass(group.artifacts[0]?.connector)}`}
        aria-label={`Activity for ${group.label}`}
      >
        <div class="source-head">
          <strong class="source-title">{group.label}</strong>
          <span
            >{group.artifacts.length} event{group.artifacts.length === 1
              ? ""
              : "s"}</span
          >
        </div>

        {#each group.artifacts as artifact (artifact.id)}
          {@const selected = selectedArtifactID === artifact.id}
          {@const link = artifactLink(artifact)}
          {@const summary = activityEventSummary(artifact)}
          {@const basketItem = artifactBasketItem(artifact)}
          <article
            class:selected
            class={connectorClass(artifact.connector)}
          >
            <button
              type="button"
              class="activity-event"
              aria-expanded={selected}
              on:click={() => toggleArtifact(artifact)}
            >
              <div class="activity-meta">
                <span>{artifactOrigin(artifact)}</span>
                <small
                  >{artifact.connector} | {artifactProvider(artifact)}</small
                >
              </div>
              <strong class="source-title">{artifact.title || artifact.source_uri}</strong>
              <p>{selected ? previewText(summary.preview, 720) : previewText(summary.preview, 220)}</p>
              <div class="activity-foot">
                <small>{artifact.event_type}</small>
                <small>{formatTime(artifact.ingested_at)}</small>
              </div>
            </button>

            {#if selected}
              <div class="activity-detail">
                {#if basketItem}
                  <div class="detail-actions">
                    <button
                      type="button"
                      on:click={() =>
                        onAskEvidence(
                          askChatPromptForEvidence(
                            basketItem.connector,
                            basketItem.uri,
                            basketItem.label,
                          ),
                        )}
                    >
                      Ask about this
                    </button>
                    <button
                      type="button"
                      disabled={basketHas(basketItem)}
                      on:click={() => onPinEvidence(basketItem)}
                    >
                      {basketHas(basketItem) ? "Pinned for analysis" : "Pin for analysis"}
                    </button>
                  </div>
                {/if}
                <div class="detail-copy">
                  <strong>Event summary</strong>
                  <SafeMarkdownBlock
                    text={summary.detailText}
                    emptyText="No summary text was saved for this event."
                    variant="detail"
                  />
                </div>
                {#if summary.facts.length}
                  <div class="detail-list">
                    <strong>Key lines</strong>
                    <SafeMarkdownBlock
                      text={markdownBulletList(summary.facts)}
                      variant="detail"
                    />
                  </div>
                {/if}
                {#if summary.links.length}
                  <div class="detail-list">
                    <strong>Links</strong>
                    {#each summary.links as item}
                      <a href={item} target="_blank" rel="noreferrer">{item}</a>
                    {/each}
                  </div>
                {/if}
                <dl>
                  {#each artifactDetailRows(artifact) as [label, value]}
                    <div>
                      <dt>{label}</dt>
                      <dd>{value}</dd>
                    </div>
                  {/each}
                </dl>
                {#if link}
                  <a href={link} target="_blank" rel="noreferrer">Open source</a
                  >
                {/if}
                {#if summary.rawText}
                  <details class="raw-event">
                    <summary>Raw event text</summary>
                    <SafeMarkdownBlock
                      text={previewMarkdownText(summary.rawText, 1800)}
                      variant="detail"
                    />
                  </details>
                {/if}
              </div>
            {/if}
          </article>
        {/each}
      </section>
    {/each}
  {:else}
    <div class="empty-state">
      <br />
      <strong>No activity loaded</strong>
      <p>
        {recentArtifacts.length
          ? "No events match the selected time window."
          : "Ask chat about recent activity or run analysis after connecting sources."}
      </p>
    </div>
  {/if}
</div>

<style>
  button,
  select,
  input {
    font: inherit;
  }

  .activity-view {
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 0;
    overflow: auto;
    scrollbar-width: none;
    padding: 0 0 14px;
  }

  .activity-view::-webkit-scrollbar {
    display: none;
  }

  .source-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .activity-filters {
    position: sticky;
    top: 0;
    z-index: 1;
    display: flex;
    align-items: flex-end;
    gap: 14px;
    border-bottom: 1px solid #d7d2c8;
    background: #ebe8e0;
    padding: 4px 16px 8px;
  }

  .filter-grid {
    flex: 1 1 auto;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(132px, 1fr));
    gap: 10px 14px;
    min-width: 0;
  }

  .filter-actions {
    display: flex;
    align-items: flex-end;
    justify-content: flex-end;
    gap: 14px;
    flex: 0 0 auto;
    padding-bottom: 1px;
    white-space: nowrap;
  }

  .activity-filters label,
  .activity-filters span {
    display: block;
    min-width: 0;
  }

  .activity-filters span {
    color: #8a8678;
    font-size: 11px;
    text-transform: uppercase;
  }

  .activity-filters select,
  .activity-filters input {
    width: 100%;
    min-width: 0;
    border: 0;
    border-bottom: 1px solid #bdb7a8;
    border-radius: 0;
    background: transparent;
    color: #1c1b18;
    padding: 5px 0;
    outline: none;
  }

  .activity-filters select:focus,
  .activity-filters input:focus {
    border-bottom-color: #1c1b18;
  }

  .activity-filters button,
  .detail-actions button {
    border: 0;
    border-bottom: 1px solid #bdb7a8;
    border-radius: 0;
    background: transparent;
    color: #1c1b18;
    cursor: pointer;
    font-weight: 700;
    padding: 6px 0;
  }

  .activity-filters button:hover,
  .detail-actions button:hover:not(:disabled) {
    border-bottom-color: #1c1b18;
  }

  .detail-actions button:disabled {
    cursor: default;
    opacity: 0.45;
  }

  .cleanup-action {
    border: 0;
    border-bottom: 1px solid #bdb7a8;
    border-radius: 0;
    background: transparent;
    color: #1c1b18;
    cursor: pointer;
    font-weight: 700;
    padding: 6px 0;
  }

  .cleanup-action {
    color: #8a3b27;
  }

  .cleanup-message {
    border-bottom: 1px solid #d7d2c8;
    padding: 12px 16px;
  }

  .cleanup-message {
    margin: 5px 0 0;
    color: #625f55;
    font-size: 12px;
    line-height: 1.45;
  }

  .source-group {
    border-bottom: 1px solid #d7d2c8;
    padding: 12px 0 4px;
  }

  .source-head {
    color: #1c1b18;
    padding: 4px 16px 8px;
  }

  .source-title {
    color: #1c1b18;
    overflow-wrap: anywhere;
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

  .source-head span,
  .activity-filters label > span {
    color: #8a8678;
    font-size: 11px;
    text-transform: uppercase;
  }

  .activity-view article,
  .empty-state {
    border-top: 1px solid #e4ded2;
    padding: 0;
  }

  .empty-state {
    padding: 16px;
  }

  .activity-view article.selected {
    background: transparent;
  }

  .activity-event {
    display: block;
    width: 100%;
    border: 0;
    background: transparent;
    color: inherit;
    padding: 12px 16px;
    text-align: left;
    cursor: pointer;
  }

  .activity-view span,
  .activity-view small {
    display: block;
    color: #8a8678;
    font-size: 11px;
    text-transform: uppercase;
  }

  .activity-view strong,
  .empty-state strong {
    display: block;
    margin-top: 0;
  }

  .activity-view p,
  .empty-state p {
    margin: 6px 0 0;
    color: #5f5b50;
    line-height: 1.45;
  }

  .activity-event p {
    overflow-wrap: anywhere;
  }

  .activity-meta {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }

  .activity-meta small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .activity-foot {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin-top: 8px;
  }

  .activity-detail {
    display: grid;
    gap: 12px;
    border-top: 1px solid #d7d2c8;
    padding: 16px;
  }

  @media (max-width: 920px) {
    .activity-filters {
      flex-wrap: wrap;
    }

    .filter-actions {
      flex: 1 1 100%;
    }
  }

  .detail-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 14px;
    border-bottom: 1px solid #e4ded2;
    padding-bottom: 10px;
  }

  .detail-copy {
    display: grid;
    gap: 7px;
  }

  .detail-list {
    display: grid;
    gap: 7px;
    border-top: 1px solid #e4ded2;
    padding-top: 10px;
  }

  .activity-detail dl {
    display: grid;
    gap: 0;
    margin: 0;
    border-top: 1px solid #e4ded2;
  }

  .activity-detail dl > div {
    display: grid;
    grid-template-columns: 120px minmax(0, 1fr);
    gap: 12px;
    border-bottom: 1px solid #e4ded2;
    padding: 7px 0;
  }

  .activity-detail dt,
  .activity-detail dd {
    margin: 0;
    overflow-wrap: anywhere;
    font-size: 11px;
  }

  .activity-detail dt {
    color: #8a8678;
    text-transform: uppercase;
  }

  .activity-detail a {
    width: max-content;
    max-width: 100%;
    border-bottom: 1px solid #bdb7a8;
    color: #1c1b18;
    font-size: 12px;
    font-weight: 700;
    text-decoration: none;
    overflow-wrap: anywhere;
  }

  .activity-detail a:hover {
    border-bottom-color: #1c1b18;
  }

  .raw-event {
    display: grid;
    gap: 8px;
    border-top: 1px solid #e4ded2;
    padding-top: 10px;
  }

  .raw-event summary {
    cursor: pointer;
    font-size: 12px;
    font-weight: 700;
  }

  @media (max-width: 640px) {
    .filter-grid {
      grid-template-columns: 1fr;
    }

    .filter-actions {
      justify-content: flex-start;
    }

    .activity-detail dl > div {
      grid-template-columns: 1fr;
      gap: 4px;
    }
  }
</style>
