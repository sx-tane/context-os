<script lang="ts">
  import type { Artifact } from "$lib/types";
  import {
    activityFilterLabel,
    artifactDetailRows,
    artifactLink,
    artifactOrigin,
    artifactProvider,
    formatTime,
    filterArtifactsByTime,
    groupArtifactsBySource,
    normalizeActivityTimeFilter,
    previewText,
    type ActivityTimeFilter,
  } from "$lib/findings/viewModel";

  export let recentArtifacts: Artifact[] = [];
  export let onCleanupNoisyLiveEvidence: () => Promise<string> = async () => "";

  const filterStorageKey = "contextos_activity_time_filter";
  const filters: ActivityTimeFilter[] = ["24h", "7d", "30d", "all"];

  let timeFilter: ActivityTimeFilter = "7d";
  let selectedArtifactID = "";
  let cleanupConfirmOpen = false;
  let cleanupRunning = false;
  let cleanupMessage = "";

  $: visibleArtifacts = filterArtifactsByTime(recentArtifacts, timeFilter);
  $: sourceGroups = groupArtifactsBySource(visibleArtifacts);

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

  function toggleArtifact(artifact: Artifact) {
    selectedArtifactID = selectedArtifactID === artifact.id ? "" : artifact.id;
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
  <div class="activity-toolbar">
    <div>
      <strong>Activity</strong>
      <span>{visibleArtifacts.length} of {recentArtifacts.length} events</span>
    </div>
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
    <button
      type="button"
      class="cleanup-action"
      on:click={() => {
        cleanupConfirmOpen = true;
        cleanupMessage = "";
      }}
    >
      Clean noisy live evidence
    </button>
  </div>

  {#if cleanupConfirmOpen}
    <section class="cleanup-confirm" aria-label="Confirm live evidence cleanup">
      <div>
        <strong>Clean noisy live evidence?</strong>
        <p>
          This removes old live chat Activity rows created from duplicate full
          answers, URL path fragments, or generic terms. It does not run
          automatically.
        </p>
      </div>
      <div class="cleanup-buttons">
        <button type="button" on:click={confirmCleanup} disabled={cleanupRunning}>
          {cleanupRunning ? "Cleaning..." : "Clean"}
        </button>
        <button
          type="button"
          on:click={() => (cleanupConfirmOpen = false)}
          disabled={cleanupRunning}
        >
          Cancel
        </button>
      </div>
    </section>
  {/if}

  {#if cleanupMessage}
    <p class="cleanup-message">{cleanupMessage}</p>
  {/if}

  {#if sourceGroups.length}
    {#each sourceGroups as group (group.key)}
      <section class="source-group" aria-label={`Activity for ${group.label}`}>
        <div class="source-head">
          <strong>{group.label}</strong>
          <span
            >{group.artifacts.length} event{group.artifacts.length === 1
              ? ""
              : "s"}</span
          >
        </div>

        {#each group.artifacts as artifact (artifact.id)}
          {@const selected = selectedArtifactID === artifact.id}
          {@const link = artifactLink(artifact)}
          <article class:selected>
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
              <strong>{artifact.title || artifact.source_uri}</strong>
              <p>
                {previewText(
                  artifact.preview || artifact.body,
                  selected ? 720 : 220,
                )}
              </p>
              <div class="activity-foot">
                <small>{artifact.event_type}</small>
                <small>{formatTime(artifact.ingested_at)}</small>
              </div>
            </button>

            {#if selected}
              <div class="activity-detail">
                <div class="detail-copy">
                  <strong>What this event is related to</strong>
                  <p>
                    {previewText(artifact.body || artifact.preview, 1200) ||
                      "No body text was saved for this event."}
                  </p>
                </div>
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
  select {
    font: inherit;
  }

  .activity-view {
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 0;
    overflow: auto;
    scrollbar-width: none;
    padding: 14px 0;
  }

  .activity-view::-webkit-scrollbar {
    display: none;
  }

  .activity-toolbar,
  .source-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .activity-toolbar {
    position: sticky;
    top: 0;
    z-index: 1;
    border-bottom: 1px solid #d7d2c8;
    background: #ebe8e0;
    padding: 10px 0;
  }

  .activity-toolbar strong,
  .activity-toolbar span,
  .activity-toolbar label {
    display: block;
  }

  .activity-toolbar label {
    min-width: 132px;
  }

  .activity-toolbar select {
    width: 100%;
    border: 0;
    border-bottom: 1px solid #bdb7a8;
    border-radius: 0;
    background: transparent;
    color: #1c1b18;
    padding: 5px 0;
    font-weight: 700;
  }

  .cleanup-action,
  .cleanup-buttons button {
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

  .cleanup-confirm,
  .cleanup-message {
    border-bottom: 1px solid #d7d2c8;
    padding: 12px 0;
  }

  .cleanup-confirm {
    display: flex;
    justify-content: space-between;
    gap: 16px;
  }

  .cleanup-confirm p,
  .cleanup-message {
    margin: 5px 0 0;
    color: #625f55;
    font-size: 12px;
    line-height: 1.45;
  }

  .cleanup-buttons {
    display: flex;
    align-items: flex-start;
    gap: 10px;
  }

  .cleanup-buttons button:disabled {
    cursor: wait;
    opacity: 0.5;
  }

  .source-group {
    border-bottom: 1px solid #d7d2c8;
    padding: 12px 0 4px;
  }

  .source-head {
    color: #1c1b18;
    padding: 4px 0 8px;
  }

  .source-head strong {
    overflow-wrap: anywhere;
  }

  .source-head span,
  .activity-toolbar span,
  .activity-toolbar label > span {
    color: #8a8678;
    font-size: 11px;
    text-transform: uppercase;
  }

  .activity-view article,
  .empty-state {
    border-top: 1px solid #e4ded2;
    padding: 0;
  }

  .activity-view article.selected {
    background: rgba(248, 246, 239, 0.48);
  }

  .activity-event {
    display: block;
    width: 100%;
    border: 0;
    background: transparent;
    color: inherit;
    padding: 12px 0;
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
    padding: 12px 0 14px;
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
    border-bottom: 1px solid #bdb7a8;
    color: #1c1b18;
    font-size: 12px;
    font-weight: 700;
    text-decoration: none;
  }

  .activity-detail a:hover {
    border-bottom-color: #1c1b18;
  }

  @media (max-width: 640px) {
    .activity-toolbar {
      align-items: stretch;
      flex-direction: column;
    }

    .cleanup-confirm {
      flex-direction: column;
    }

    .activity-detail dl > div {
      grid-template-columns: 1fr;
      gap: 4px;
    }
  }
</style>
