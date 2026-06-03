<script lang="ts">
    import type { ChatMessage } from "$lib/types";
    import {
        artifactLink,
        artifactSourceLabel,
        findingDescription,
        findingImpact,
        findingRecommendedAction,
        findingSummary,
        formatTime,
        messageLines,
        severityLabel,
    } from "$lib/findingsViewModel";

    export let messages: ChatMessage[] = [];
    export let hasSources = false;
    export let busy = false;
    export let command = "";
    export let onClear: () => void = () => {};
    export let onSubmit: () => void | Promise<void> = () => {};
</script>

<section class="chat-card">
    <div class="chat-head">
        <div>
            <strong>Report Agent</strong>
            <span>{hasSources ? "Ask against selected sources" : "Connect sources before asking"}</span>
        </div>
        <button type="button" on:click={onClear}>Clear</button>
    </div>

    <div class="messages" aria-live="polite">
        {#if messages.length === 0}
            <article class="message assistant">
                <span>CONTEXT-OS</span>
                <p>{hasSources ? "Ask about Slack messages, GitHub PRs, Jira tickets, docs, findings, or recent activity." : "Connect GitHub repos, Slack channels, or docs first."}</p>
            </article>
        {:else}
            {#each messages as message (message.id)}
                <article class="message" class:user={message.role === "user"}>
                    <span>{message.role === "user" ? "YOU" : "CONTEXT-OS"}</span>
                    <div class="message-body">
                        {#each messageLines(message.loading ? message.text || "Working..." : message.text) as line}
                            {#if line.kind === "blank"}
                                <div class="message-gap"></div>
                            {:else if line.kind === "heading"}
                                <h4>{line.text}</h4>
                            {:else if line.kind === "number"}
                                <p class="number-line">{line.text}</p>
                            {:else if line.kind === "bullet"}
                                <p class="bullet-line">{line.text}</p>
                            {:else}
                                <p>{line.text}</p>
                            {/if}
                        {/each}
                    </div>
                    {#if message.card?.chatResult?.artifacts?.length}
                        <details>
                            <summary>{message.card.chatResult.artifact_count} evidence items</summary>
                            {#each message.card.chatResult.artifacts.slice(0, 5) as artifact (artifact.id)}
                                {@const link = artifactLink(artifact)}
                                <div class="evidence-item">
                                    <div class="evidence-meta">
                                        <span>{artifact.connector}</span>
                                        <small>{formatTime(artifact.ingested_at)}</small>
                                    </div>
                                    <strong>{artifactSourceLabel(artifact)}</strong>
                                    <div class="evidence-source-row">
                                        {#if link}
                                            <a href={link} target="_blank" rel="noreferrer">Open source</a>
                                        {:else}
                                            <span>{artifact.source_uri || "Stored local source"}</span>
                                        {/if}
                                    </div>
                                </div>
                            {/each}
                        </details>
                    {/if}
                    {#if message.card?.findingsResult?.mismatches?.length}
                        <details>
                            <summary>{message.card.findingsResult.mismatch_count ?? message.card.findingsResult.mismatches.length} findings</summary>
                            {#each message.card.findingsResult.mismatches.slice(0, 5) as mismatch}
                                <div class="evidence-item">
                                    <div class="finding-preview-head">
                                        <span>{severityLabel(mismatch.severity)}</span>
                                        <strong>{findingSummary(mismatch)}</strong>
                                    </div>
                                    <p>{findingDescription(mismatch)}</p>
                                    {#if findingImpact(mismatch)}
                                        <p><b>Impact:</b> {findingImpact(mismatch)}</p>
                                    {/if}
                                    {#if findingRecommendedAction(mismatch)}
                                        <p><b>Recommended:</b> {findingRecommendedAction(mismatch)}</p>
                                    {/if}
                                </div>
                            {/each}
                        </details>
                    {/if}
                </article>
            {/each}
        {/if}
    </div>

    <form class="composer" on:submit|preventDefault={onSubmit}>
        <input
            bind:value={command}
            disabled={busy || !hasSources}
            placeholder={hasSources ? "Ask about PRs, Slack threads, findings, or recent activity..." : "Connect sources first..."}
        />
        <button class="send-icon" aria-label="Send message" title="Send" disabled={busy || !hasSources || command.trim() === ""}>↑</button>
    </form>
</section>

<style>
    button,
    input {
        font: inherit;
    }

    button {
        cursor: pointer;
    }

    .chat-card {
        flex: 1 1 auto;
        min-height: 280px;
        display: grid;
        grid-template-rows: auto minmax(0, 1fr) auto;
        overflow: hidden;
        background: transparent;
    }

    .chat-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        border-bottom: 1px solid #d7d2c8;
        padding: 4px 0 12px;
    }

    .chat-head strong,
    .chat-head span {
        display: block;
    }

    .chat-head span,
    .message > span {
        letter-spacing: 0.05em;
    }

    .chat-head span {
        margin-top: 3px;
        color: #8a8678;
        font-size: 12px;
    }

    .chat-head button,
    .composer button {
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background-color: transparent;
        background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
        background-position: 100% 0;
        background-size: 200% 100%;
        color: #1c1b18;
        font-weight: 700;
        transition:
            background-position 0.18s ease,
            color 0.15s,
            border-color 0.15s,
            opacity 0.15s;
    }

    .chat-head button {
        padding: 7px 12px;
    }

    .chat-head button:hover,
    .composer button:hover:not(:disabled) {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .composer button:disabled {
        cursor: not-allowed;
        opacity: 0.42;
    }

    .messages {
        min-height: 0;
        display: flex;
        flex-direction: column;
        gap: 12px;
        overflow: auto;
        scrollbar-width: none;
        overscroll-behavior: contain;
        padding: 16px;
    }

    .messages::-webkit-scrollbar {
        display: none;
    }

    .message {
        width: min(680px, 90%);
        border-radius: 14px;
        background: transparent;
        padding: 4px 0;
        line-height: 1.5;
    }

    .message.user {
        align-self: flex-end;
        color: #1c1b18;
        padding: 4px 0;
        text-align: right;
    }

    .message > span {
        display: block;
        margin-bottom: 6px;
        color: #8a8678;
        font-size: 12px;
    }

    .message p {
        margin: 0;
    }

    .message-body {
        display: grid;
        gap: 6px;
    }

    .message-body h4 {
        margin: 10px 0 2px;
        font-size: 13px;
        color: #28261f;
    }

    .message-body h4:first-child {
        margin-top: 0;
    }

    .message-body .number-line {
        margin-top: 8px;
        font-weight: 700;
    }

    .message-body .bullet-line {
        position: relative;
        padding-left: 14px;
    }

    .message-body .bullet-line::before {
        content: "";
        position: absolute;
        left: 0;
        top: 0.72em;
        width: 5px;
        height: 5px;
        border-radius: 50%;
        background: #8a8678;
    }

    .message-gap {
        height: 4px;
    }

    details {
        margin-top: 12px;
        border-top: 1px solid #d7d2c8;
        padding-top: 10px;
    }

    summary {
        cursor: pointer;
        font-weight: 700;
    }

    .evidence-item {
        margin-top: 10px;
        border: 0;
        border-top: 1px solid #d7d2c8;
        border-radius: 0;
        background: transparent;
        padding: 10px;
    }

    .evidence-meta,
    .evidence-source-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        min-width: 0;
    }

    .evidence-meta span,
    .finding-preview-head span {
        color: #d85d3f;
        font-size: 11px;
        font-weight: 700;
        text-transform: uppercase;
    }

    .evidence-item strong {
        display: block;
        margin: 7px 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .evidence-source-row a {
        color: #1f5f8b;
        font-weight: 700;
        text-decoration: none;
    }

    .evidence-source-row a:hover {
        color: #1c1b18;
        text-decoration: underline;
    }

    .evidence-source-row span {
        min-width: 0;
        overflow: hidden;
        color: #8a8678;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 12px;
    }

    .finding-preview-head {
        display: flex;
        align-items: baseline;
        gap: 10px;
        min-width: 0;
    }

    .finding-preview-head span {
        flex: 0 0 auto;
        letter-spacing: 0.04em;
    }

    .finding-preview-head strong {
        min-width: 0;
        overflow-wrap: anywhere;
    }

    .composer {
        display: grid;
        grid-template-columns: 1fr auto;
        gap: 10px;
        padding: 12px;
        border-top: 1px solid #d7d2c8;
    }

    .composer input {
        min-width: 0;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background: transparent;
        padding: 11px 12px;
        outline: none;
    }

    .composer input:focus {
        border-bottom-color: #1c1b18;
    }

    .composer button {
        width: 44px;
        min-width: 44px;
        padding: 0;
        color: #1c1b18;
        font-size: 18px;
        font-weight: 700;
        line-height: 1;
    }

    small {
        color: #8a8678;
    }
</style>
