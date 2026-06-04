<script lang="ts">
    import type { Artifact, FindingsResult, WorkspaceStatus } from "$lib/types";
    import type {
        FindingsInsightState,
        InsightStatus,
    } from "$lib/insights/status";
    import {
        findingDescription,
        findingDetectedTime,
        findingEvidenceTime,
        findingImpact,
        findingRecommendedAction,
        findingSummary,
        severityLabel,
    } from "$lib/findings/viewModel";

    export let lastFindings: FindingsResult | null = null;
    export let lastAnalysisAt = "";
    export let recentArtifacts: Artifact[] = [];
    export let readySourceCount = 0;
    export let workspaceStatus: WorkspaceStatus | null = null;
    export let hasSources = false;
    export let insightStatus: InsightStatus | null = null;

    function shouldShowStatusNote(status: InsightStatus | null) {
        return status !== null && status.findingsState !== "current";
    }

    function isAttentionState(state: FindingsInsightState | undefined) {
        return state === "stale" || state === "no_concrete_sources";
    }

    function zeroFindingTitle(status: InsightStatus | null) {
        if (status?.findingsState === "stale") return "Findings stale";
        if (status?.findingsState === "no_concrete_sources") {
            return "No concrete analysis sources";
        }
        return "Analysis ran, no mismatch signals detected";
    }

    function zeroFindingMessage(status: InsightStatus | null) {
        return status?.findingsMessage ??
            "Findings are current for the latest analyzed evidence.";
    }

    function emptyFindingTitle(status: InsightStatus | null) {
        if (status?.findingsState === "no_concrete_sources") {
            return "No concrete analysis sources";
        }
        if (status?.findingsState === "stale") return "Findings stale";
        if (status?.findingsState === "not_run" && status.hasGraphContext) {
            return "Graph has context, findings not run yet";
        }
        return hasSources ? "No findings yet" : "Connect sources to unlock findings";
    }

    function emptyFindingMessage(status: InsightStatus | null) {
        return status?.findingsMessage ??
            (hasSources
                ? "Run analysis across selected sources to surface mismatches and delivery risks."
                : "Select GitHub repos, Slack channels, or docs first.");
    }
</script>

<div class="findings-view">
    {#if lastFindings?.mismatches?.length}
        {#if shouldShowStatusNote(insightStatus)}
            <div
                class="findings-state-row"
                class:attention={isAttentionState(insightStatus?.findingsState)}
            >
                <span>{insightStatus?.findingsLabel}</span>
                <p>{insightStatus?.findingsMessage}</p>
                {#if insightStatus?.chatOnlySources.length}
                    <div class="chat-only-scopes">
                        <small>Skipped chat-only scopes</small>
                        {#each insightStatus.chatOnlySources as source}
                            <code>{source.connector}:{source.uri}</code>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}
        {#each lastFindings.mismatches.slice(0, 6) as mismatch}
            <article>
                <div class="finding-title-row">
                    <span>{severityLabel(mismatch.severity)}</span>
                    <strong>{findingSummary(mismatch)}</strong>
                </div>
                <div class="finding-time-row">
                    <small>Detected: {findingDetectedTime(lastAnalysisAt)}</small>
                    <small>Evidence: {findingEvidenceTime(recentArtifacts, lastAnalysisAt)}</small>
                </div>
                <div class="finding-copy">
                    <p>{findingDescription(mismatch)}</p>
                    {#if findingImpact(mismatch)}
                        <p><b>Impact:</b> {findingImpact(mismatch)}</p>
                    {/if}
                </div>
                {#if findingRecommendedAction(mismatch)}
                    <div class="finding-action">
                        <small>Recommended action</small>
                        <p>{findingRecommendedAction(mismatch)}</p>
                    </div>
                {/if}
            </article>
        {/each}
    {:else if lastFindings}
        <div class="empty-state">
            <strong>{zeroFindingTitle(insightStatus)}</strong>
            <p>{zeroFindingMessage(insightStatus)}</p>
            <p>Detected: {findingDetectedTime(lastAnalysisAt)}</p>
            <p>Sources: {lastFindings.uri ?? readySourceCount}. Events: {lastFindings.event_count ?? workspaceStatus?.event_count ?? 0}. Entities: {lastFindings.entity_count ?? workspaceStatus?.entity_count ?? 0}.</p>
            {#if insightStatus?.chatOnlySources.length}
                <div class="chat-only-scopes">
                    <small>Skipped chat-only scopes</small>
                    {#each insightStatus.chatOnlySources as source}
                        <code>{source.connector}:{source.uri}</code>
                    {/each}
                </div>
            {/if}
        </div>
    {:else}
        <div class="empty-state">
            <strong>{emptyFindingTitle(insightStatus)}</strong>
            <p>{emptyFindingMessage(insightStatus)}</p>
            {#if insightStatus?.chatOnlySources.length}
                <div class="chat-only-scopes">
                    <small>Skipped chat-only scopes</small>
                    {#each insightStatus.chatOnlySources as source}
                        <code>{source.connector}:{source.uri}</code>
                    {/each}
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .findings-view {
        min-height: 0;
        display: flex;
        flex-direction: column;
        gap: 10px;
        overflow: auto;
        padding: 14px 0;
    }

    .findings-view article,
    .findings-state-row,
    .empty-state {
        border-bottom: 1px solid #d7d2c8;
        padding: 14px 0;
    }

    .findings-view span {
        display: block;
        color: #8a8678;
        font-size: 11px;
        text-transform: uppercase;
    }

    .finding-title-row {
        display: flex;
        align-items: baseline;
        gap: 10px;
        min-width: 0;
    }

    .finding-title-row span {
        flex: 0 0 auto;
        color: #d85d3f;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.04em;
    }

    .finding-title-row strong {
        min-width: 0;
        overflow-wrap: anywhere;
    }

    .finding-time-row {
        display: flex;
        flex-wrap: wrap;
        gap: 6px 14px;
        margin-top: 6px;
        padding-bottom: 8px;
        border-bottom: 1px solid rgba(215, 210, 200, 0.62);
    }

    .finding-copy,
    .finding-action {
        margin-top: 10px;
    }

    .findings-state-row {
        padding-top: 0;
    }

    .findings-state-row.attention span {
        color: #b5523a;
        font-weight: 700;
    }

    .finding-action {
        padding-left: 10px;
        border-left: 2px solid #d7d2c8;
    }

    .finding-action small {
        display: block;
        margin-bottom: 2px;
        font-weight: 700;
        letter-spacing: 0.03em;
        text-transform: uppercase;
    }

    .findings-view strong,
    .empty-state strong {
        display: block;
        margin-top: 0;
    }

    .findings-view p,
    .empty-state p {
        margin: 6px 0 0;
        color: #5f5b50;
        line-height: 1.45;
    }

    .chat-only-scopes {
        display: flex;
        flex-wrap: wrap;
        gap: 6px 8px;
        margin-top: 10px;
        padding-top: 8px;
        border-top: 1px solid rgba(215, 210, 200, 0.62);
    }

    .chat-only-scopes small {
        flex: 1 0 100%;
        font-weight: 700;
        text-transform: uppercase;
    }

    .chat-only-scopes code {
        max-width: 100%;
        border-bottom: 1px solid #d7d2c8;
        padding-bottom: 2px;
        color: #28261f;
        overflow-wrap: anywhere;
        font: inherit;
    }

    small {
        color: #8a8678;
    }
</style>
