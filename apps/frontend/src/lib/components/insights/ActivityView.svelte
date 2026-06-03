<script lang="ts">
    import type { Artifact } from "$lib/types";
    import {
        artifactOrigin,
        artifactProvider,
        formatTime,
        previewText,
    } from "$lib/findingsViewModel";

    export let recentArtifacts: Artifact[] = [];
</script>

<div class="activity-view">
    {#if recentArtifacts.length}
        {#each recentArtifacts.slice(0, 8) as artifact (artifact.id)}
            <article>
                <div class="activity-meta">
                    <span>{artifactOrigin(artifact)}</span>
                    <small>{artifact.connector} | {artifactProvider(artifact)}</small>
                </div>
                <strong>{artifact.title || artifact.source_uri}</strong>
                <p>{previewText(artifact.preview)}</p>
                <small>{artifact.source_uri}</small>
                <small>{formatTime(artifact.ingested_at)}</small>
            </article>
        {/each}
    {:else}
        <div class="empty-state">
            <strong>No activity loaded</strong>
            <p>Ask chat about recent activity or run analysis after connecting sources.</p>
        </div>
    {/if}
</div>

<style>
    .activity-view {
        min-height: 0;
        display: flex;
        flex-direction: column;
        gap: 10px;
        overflow: auto;
        padding: 14px 0;
    }

    .activity-view article,
    .empty-state {
        border-bottom: 1px solid #d7d2c8;
        padding: 14px 0;
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
</style>
