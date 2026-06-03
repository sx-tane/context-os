<script lang="ts">
    import { isNearBottom, localDBStatusLine } from "$lib/chatController";
    import type { ChatMessage, ChatQueryResult } from "$lib/types";
    import InlineText from "$lib/components/chat/InlineText.svelte";
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

    let messagesEl: HTMLDivElement | null = null;
    let composerForm: HTMLFormElement | null = null;
    let composerTextarea: HTMLTextAreaElement | null = null;
    let stickToBottom = true;
    let expandedStreams: Record<string, boolean> = {};
    let lastMessageSignature = "";

    $: messageSignature = messages
        .map((message) => [
            message.id,
            message.text,
            message.loading ? "loading" : "done",
            message.stream?.status ?? "",
            message.stream?.latestLine ?? "",
            message.stream?.lines.length ?? 0,
        ].join(":"))
        .join("|");

    $: if (messagesEl && messageSignature !== lastMessageSignature) {
        lastMessageSignature = messageSignature;
        const shouldScroll = stickToBottom;
        if (shouldScroll) {
            requestAnimationFrame(scrollMessagesToBottom);
        }
    }

    function queryProviderLabel(provider?: string) {
        return provider === "codex" ? "Live Codex" : "Local DB";
    }

    function querySourceLabel(result?: ChatQueryResult) {
        if (!result) return "";
        const parts = [];
        if (result.connector) parts.push(result.connector);
        if (result.source_uri) parts.push(result.source_uri);
        return parts.join(" · ");
    }

    function streamStatusLabel(message: ChatMessage) {
        if (message.stream?.status === "complete") return "Complete";
        if (message.stream?.status === "error") return "Error";
        return "Running";
    }

    function streamExpanded(message: ChatMessage) {
        return expandedStreams[message.id] ?? message.stream?.expanded ?? false;
    }

    function toggleStream(message: ChatMessage) {
        expandedStreams = {
            ...expandedStreams,
            [message.id]: !streamExpanded(message),
        };
    }

    function traceSummary(result: ChatQueryResult, message: ChatMessage) {
        const pieces = [
            `Provider: ${queryProviderLabel(result.provider)}`,
        ];
        if (result.connector) pieces.push(`Connector: ${result.connector}`);
        if (result.source_uri) pieces.push(`Source: ${result.source_uri}`);
        if (
            message.stream?.summary &&
            message.stream.summary !== result.answer &&
            message.stream.summary !== result.summary &&
            !message.stream.summary.startsWith("Local DB:")
        ) {
            pieces.push(`Stream: ${message.stream.summary}`);
        }
        pieces.push(`Artifacts: ${result.artifact_count ?? result.artifacts?.length ?? 0}`);
        return pieces.join(" · ");
    }

    function sourceTraceLabel(result: ChatQueryResult) {
        if (result.artifacts?.length) return "Source trace";
        if (result.provider === "codex") return "Live source trace";
        return "Source trace";
    }

    function handleMessagesScroll() {
        if (!messagesEl) return;
        stickToBottom = isNearBottom(messagesEl);
    }

    function scrollMessagesToBottom() {
        if (!messagesEl) return;
        messagesEl.scrollTop = messagesEl.scrollHeight;
        stickToBottom = true;
    }

    function handleComposerKeydown(event: KeyboardEvent) {
        if (event.key !== "Enter" || (!event.ctrlKey && !event.metaKey)) {
            return;
        }
        event.preventDefault();
        composerForm?.requestSubmit();
    }

    function resizeComposer() {
        if (!composerTextarea) return;
        composerTextarea.style.height = "auto";
        composerTextarea.style.height = `${Math.min(composerTextarea.scrollHeight, 132)}px`;
    }

    $: if (composerTextarea) {
        command;
        requestAnimationFrame(resizeComposer);
    }
</script>

<section class="chat-card">
    <div class="chat-head">
        <div>
            <strong>Report Agent</strong>
            <span>{hasSources ? "Ask against selected sources" : "Connect sources before asking"}</span>
        </div>
        <button type="button" on:click={onClear}>Clear</button>
    </div>

    <div
        class="messages"
        aria-live="polite"
        bind:this={messagesEl}
        on:scroll={handleMessagesScroll}
    >
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
                        {#each messageLines(message.text || (message.loading && !message.stream ? "Working..." : "")) as line}
                            {#if line.kind === "blank"}
                                <div class="message-gap"></div>
                            {:else if line.kind === "heading"}
                                <h4><InlineText text={line.text} /></h4>
                            {:else if line.kind === "number"}
                                <p class="number-line"><InlineText text={line.text} /></p>
                            {:else if line.kind === "bullet"}
                                <p class="bullet-line"><InlineText text={line.text} /></p>
                            {:else}
                                <p><InlineText text={line.text} /></p>
                            {/if}
                        {/each}
                    </div>
                    {#if message.stream}
                        <section
                            class="stream-panel"
                            class:running={message.stream.status === "running"}
                            class:error={message.stream.status === "error"}
                            aria-label="Live stream progress"
                        >
                            <button
                                type="button"
                                class="stream-toggle"
                                aria-expanded={streamExpanded(message)}
                                on:click={() => toggleStream(message)}
                            >
                                <span>{streamStatusLabel(message)}</span>
                                <strong>{streamExpanded(message) ? "Hide stream" : "Show stream"}</strong>
                            </button>
                            {#if !streamExpanded(message)}
                                <p class="stream-latest">{message.stream.latestLine}</p>
                                {#if message.stream.summary}
                                    <p class="stream-summary">{message.stream.summary}</p>
                                {/if}
                            {:else}
                                <div class="stream-lines">
                                    {#each message.stream.lines as line, index (`${message.id}-${index}`)}
                                        <p>{line}</p>
                                    {/each}
                                </div>
                            {/if}
                        </section>
                    {/if}
                    {#if message.card?.kind === "query" && message.card.chatResult}
                        <div
                            class="query-meta"
                            class:live={message.card.chatResult.provider === "codex"}
                        >
                            <span>{queryProviderLabel(message.card.chatResult.provider)}</span>
                            {#if querySourceLabel(message.card.chatResult)}
                                <small>{querySourceLabel(message.card.chatResult)}</small>
                            {/if}
                        </div>
                        {#if localDBStatusLine(message.card.chatResult)}
                            <p
                                class="local-db-status"
                                class:saved={message.card.chatResult.evidence_save_status === "saved"}
                                class:error={message.card.chatResult.evidence_save_status === "error"}
                            >
                                {localDBStatusLine(message.card.chatResult)}
                            </p>
                        {/if}
                        <div
                            class="source-trace"
                            class:live={message.card.chatResult.provider === "codex" && !message.card.chatResult.artifacts?.length}
                        >
                            <strong>{sourceTraceLabel(message.card.chatResult)}</strong>
                            <span>{traceSummary(message.card.chatResult, message)}</span>
                        </div>
                    {/if}
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

    <form
        class="composer"
        bind:this={composerForm}
        on:submit|preventDefault={onSubmit}
    >
        <textarea
            bind:this={composerTextarea}
            bind:value={command}
            disabled={busy || !hasSources}
            placeholder={hasSources ? "Ask about PRs, Slack threads, findings, or recent activity..." : "Connect sources first..."}
            rows="2"
            on:keydown={handleComposerKeydown}
            on:input={resizeComposer}
        ></textarea>
        <button class="send-icon" aria-label="Send message" title="Send" disabled={busy || !hasSources || command.trim() === ""}>↑</button>
    </form>
</section>

<style>
    button,
    textarea {
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

    .stream-panel {
        margin-top: 12px;
        border-top: 1px solid #d7d2c8;
        border-bottom: 1px solid #e4ded2;
        padding: 8px 0 10px;
        color: #625f55;
        font-size: 12px;
    }

    .stream-panel.running .stream-toggle span {
        color: #1f5f8b;
    }

    .stream-panel.error .stream-toggle span {
        color: #b4422a;
    }

    .stream-toggle {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        width: 100%;
        border: 0;
        border-radius: 0;
        background: transparent;
        padding: 0;
        color: inherit;
        text-align: left;
    }

    .stream-toggle span {
        color: #8a6a20;
        font-size: 11px;
        font-weight: 700;
        text-transform: uppercase;
    }

    .stream-toggle strong {
        flex: 0 0 auto;
        border-bottom: 1px solid #bdb7a8;
        color: #1c1b18;
        font-size: 11px;
    }

    .stream-toggle:hover strong {
        border-bottom-color: #1c1b18;
    }

    .stream-latest,
    .stream-summary {
        margin-top: 7px;
        overflow-wrap: anywhere;
    }

    .stream-summary {
        color: #1c1b18;
        font-weight: 700;
    }

    .stream-lines {
        display: grid;
        gap: 5px;
        margin-top: 8px;
    }

    .stream-lines p {
        color: #625f55;
        overflow-wrap: anywhere;
    }

    .query-meta {
        display: flex;
        align-items: center;
        gap: 8px;
        min-width: 0;
        margin-top: 10px;
        color: #625f55;
        font-size: 11px;
    }

    .query-meta span {
        flex: 0 0 auto;
        color: #8a6a20;
        font-weight: 700;
        text-transform: uppercase;
    }

    .query-meta.live span {
        color: #1f5f8b;
    }

    .query-meta small {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .local-db-status {
        margin-top: 7px;
        color: #625f55;
        font-size: 12px;
        font-weight: 700;
    }

    .local-db-status.saved {
        color: #27633a;
    }

    .local-db-status.error {
        color: #b4422a;
    }

    .source-trace {
        display: grid;
        gap: 4px;
        margin-top: 8px;
        border-top: 1px solid #e4ded2;
        padding-top: 8px;
        color: #625f55;
        font-size: 11px;
    }

    .source-trace strong {
        color: #8a6a20;
        text-transform: uppercase;
    }

    .source-trace.live strong {
        color: #1f5f8b;
    }

    .source-trace span {
        overflow-wrap: anywhere;
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

    .composer textarea {
        resize: none;
        min-width: 0;
        min-height: 47px;
        max-height: 132px;
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background: transparent;
        padding: 11px 12px;
        outline: none;
        line-height: 1.45;
        overflow: auto;
        scrollbar-width: none;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
    }

    .composer textarea::-webkit-scrollbar {
        display: none;
    }

    .composer textarea:focus {
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
