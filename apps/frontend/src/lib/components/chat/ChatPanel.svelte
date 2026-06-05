<script lang="ts">
    import { tick } from "svelte";
    import { composerHeight } from "$lib/chat/composer";
    import { isNearBottom, localDBStatusLine } from "$lib/chat/controller";
    import type { AnswerSection, ChatMessage, ChatQueryResult } from "$lib/types";
    import type { EvidenceBasketItem } from "$lib/workflow/types";
    import SafeMarkdownBlock from "$lib/components/ui/SafeMarkdownBlock.svelte";
    import {
        artifactLink,
        artifactSourceLabel,
        findingDescription,
        findingImpact,
        findingRecommendedAction,
        findingSummary,
        formatTime,
        markdownBulletList,
        reviewCandidateCount,
        severityLabel,
        topActionableFindings,
    } from "$lib/findings/viewModel";
    import {
        askChatPromptForEvidence,
        basketItemFromAnswerSection,
    } from "$lib/workflow/viewModel";

    export let messages: ChatMessage[] = [];
    export let hasSources = false;
    export let busy = false;
    export let canStop = false;
    export let command = "";
    export let basketItems: EvidenceBasketItem[] = [];
    export let onClear: () => void | Promise<void> = () => {};
    export let onSubmit: () => void | Promise<void> = () => {};
    export let onStop: () => void | Promise<void> = () => {};
    export let onAskEvidence: (prompt: string) => void | Promise<void> = () => {};
    export let onPinEvidence: (item: EvidenceBasketItem) => void | Promise<void> = () => {};

    let messagesEl: HTMLDivElement | null = null;
    let composerForm: HTMLFormElement | null = null;
    let composerTextarea: HTMLTextAreaElement | null = null;
    let stickToBottom = true;
    let expandedStreams: Record<string, boolean> = {};
    let lastMessageSignature = "";
    const expandedStreamLineLimit = 80;

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

    $: command, scheduleComposerResize();

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

    function visibleStreamLines(message: ChatMessage) {
        const lines = message.stream?.lines ?? [];
        return lines.slice(-expandedStreamLineLimit);
    }

    function hiddenStreamLineCount(message: ChatMessage) {
        const lines = message.stream?.lines ?? [];
        return Math.max(0, lines.length - expandedStreamLineLimit);
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

    function answerSections(result?: ChatQueryResult) {
        return result?.answer_sections?.filter((section) =>
            Boolean(
                section.source_label ||
                    section.source_uri ||
                    section.summary ||
                    section.facts?.length ||
                    section.open_items?.length ||
                    section.coding_notes?.length ||
                    section.links?.length,
            ),
        ) ?? [];
    }

    function sectionLabel(section: AnswerSection) {
        return section.source_label || section.source_uri || section.connector || "Source";
    }

    function sectionMeta(section: AnswerSection) {
        return [section.connector, section.status, section.confidence ? `${Math.round(section.confidence * 100)}%` : ""]
            .filter(Boolean)
            .join(" · ");
    }

    function sectionLink(section: AnswerSection) {
        const link = section.links?.find((item) => /^https?:\/\//i.test(item));
        if (link) return link;
        return /^https?:\/\//i.test(section.source_uri ?? "") ? section.source_uri ?? "" : "";
    }

    function sectionURLLinks(section: AnswerSection) {
        return section.links?.filter((item) => /^https?:\/\//i.test(item)) ?? [];
    }

    function sectionTextLinks(section: AnswerSection) {
        return section.links?.filter((item) => !/^https?:\/\//i.test(item)) ?? [];
    }

    function sectionBasketItem(section: AnswerSection, messageID: string) {
        return basketItemFromAnswerSection(section, messageID);
    }

    function basketHas(item: EvidenceBasketItem | null) {
        return Boolean(item && basketItems.some((existing) => existing.id === item.id));
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

    function handleMessagesScroll() {
        if (!messagesEl) return;
        stickToBottom = isNearBottom(messagesEl);
    }

    function scrollMessagesToBottom() {
        if (!messagesEl) return;
        messagesEl.scrollTop = messagesEl.scrollHeight;
        stickToBottom = true;
    }

    function scheduleComposerResize() {
        void tick().then(resizeComposer);
    }

    function resizeComposer() {
        if (!composerTextarea) return;
        const styles = window.getComputedStyle(composerTextarea);
        const maxHeight = parseFloat(styles.maxHeight) || composerTextarea.scrollHeight;
        const minHeight = parseFloat(styles.minHeight) || 0;
        composerTextarea.style.height = "auto";
        const nextHeight = composerHeight(
            composerTextarea.scrollHeight,
            maxHeight,
            minHeight,
        );
        composerTextarea.style.height = `${nextHeight}px`;
        composerTextarea.style.overflowY =
            composerTextarea.scrollHeight > nextHeight ? "auto" : "hidden";
    }

    function handleComposerKeydown(event: KeyboardEvent) {
        if (event.key !== "Enter") {
            return;
        }
        if (event.shiftKey) {
            return;
        }
        event.preventDefault();
        composerForm?.requestSubmit();
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
                <article
                    class="message"
                    class:user={message.role === "user"}
                    class:assistant={message.role !== "user"}
                >
                    <span>{message.role === "user" ? "YOU" : "CONTEXT-OS"}</span>
                    <div class="message-body">
                        <SafeMarkdownBlock
                            text={message.text || (message.loading && !message.stream ? "Working..." : "")}
                            variant={message.role === "user" ? "plain" : "message"}
                        />
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
                                    {#if hiddenStreamLineCount(message) > 0}
                                        <small>{hiddenStreamLineCount(message)} earlier stream lines hidden</small>
                                    {/if}
                                    {#each visibleStreamLines(message) as line, index (`${message.id}-${index}`)}
                                        <p>{line}</p>
                                    {/each}
                                </div>
                            {/if}
                        </section>
                    {/if}
                    {#if message.card?.kind === "query" && message.card.chatResult}
                        {#if answerSections(message.card.chatResult).length}
                            <div class="answer-sections" aria-label="Structured source answer">
                                {#each answerSections(message.card.chatResult) as section, index (`${message.id}-section-${index}`)}
                                    {@const link = sectionLink(section)}
                                    {@const basketItem = sectionBasketItem(section, message.id)}
                                    <section class={`answer-section ${connectorClass(section.connector)}`}>
                                        <div class="answer-section-head">
                                            <div>
                                                <strong class="source-title">{sectionLabel(section)}</strong>
                                                {#if sectionMeta(section)}
                                                    <span>{sectionMeta(section)}</span>
                                                {/if}
                                            </div>
                                            <div class="answer-section-actions">
                                                {#if basketItem}
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
                                                        Ask
                                                    </button>
                                                    <button
                                                        type="button"
                                                        disabled={basketHas(basketItem)}
                                                        on:click={() => onPinEvidence(basketItem)}
                                                    >
                                                        {basketHas(basketItem) ? "Pinned" : "Pin"}
                                                    </button>
                                                {/if}
                                                {#if link}
                                                    <a href={link} target="_blank" rel="noreferrer">Open</a>
                                                {/if}
                                            </div>
                                        </div>
                                        {#if section.summary}
                                            <div class="answer-section-copy">
                                                <SafeMarkdownBlock
                                                    text={section.summary}
                                                    variant="source"
                                                />
                                            </div>
                                        {/if}
                                        {#if section.facts?.length}
                                            <div class="answer-section-list">
                                                <strong>Facts</strong>
                                                <SafeMarkdownBlock
                                                    text={markdownBulletList(section.facts)}
                                                    variant="source"
                                                />
                                            </div>
                                        {/if}
                                        {#if section.open_items?.length}
                                            <div class="answer-section-list">
                                                <strong>Open items</strong>
                                                <SafeMarkdownBlock
                                                    text={markdownBulletList(section.open_items)}
                                                    variant="source"
                                                />
                                            </div>
                                        {/if}
                                        {#if section.coding_notes?.length}
                                            <div class="answer-section-list">
                                                <strong>Coding notes</strong>
                                                <SafeMarkdownBlock
                                                    text={markdownBulletList(section.coding_notes)}
                                                    variant="source"
                                                />
                                            </div>
                                        {/if}
                                        {#if section.links?.length}
                                            <div class="answer-section-list">
                                                <strong>Links</strong>
                                                {#if sectionURLLinks(section).length}
                                                    <div class="answer-link-list">
                                                        {#each sectionURLLinks(section) as item}
                                                            <a href={item} target="_blank" rel="noreferrer">{item}</a>
                                                        {/each}
                                                    </div>
                                                {/if}
                                                {#if sectionTextLinks(section).length}
                                                    <SafeMarkdownBlock
                                                        text={markdownBulletList(sectionTextLinks(section))}
                                                        variant="source"
                                                    />
                                                {/if}
                                            </div>
                                        {/if}
                                    </section>
                                {/each}
                            </div>
                        {/if}
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
                    {#if message.card?.findingsResult}
                        {@const topFindings = topActionableFindings(message.card.findingsResult, 3)}
                        <details>
                            <summary>
                                {message.card.findingsResult.mismatch_count ?? topFindings.length} top issues · {reviewCandidateCount(message.card.findingsResult)} review candidates
                            </summary>
                            {#if topFindings.length}
                                {#each topFindings as mismatch}
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
                            {:else}
                                <div class="evidence-item">
                                    <p>No top actionable issues. Dependency-only signals are under Review candidates.</p>
                                </div>
                            {/if}
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
            on:input={resizeComposer}
            on:keydown={handleComposerKeydown}
        ></textarea>
        {#if canStop}
            <button
                type="button"
                class="send-icon stop-icon"
                aria-label="Stop Codex chat"
                title="Stop Codex chat"
                on:click={onStop}
            >×</button>
        {:else}
            <button class="send-icon" aria-label="Send message" title="Send" disabled={busy || !hasSources || command.trim() === ""}>↑</button>
        {/if}
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

    .composer .stop-icon {
        border-bottom-color: #b4422a;
        color: #b4422a;
    }

    .composer .stop-icon:hover {
        border-bottom-color: #b4422a;
        background-image: linear-gradient(90deg, #b4422a 0 50%, transparent 50% 100%);
        color: #f8f6ef;
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
        width: min(760px, 94%);
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
        gap: 7px;
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

    .answer-sections {
        display: grid;
        gap: 12px;
        margin-top: 14px;
    }

    .answer-section {
        border-top: 1px solid #d7d2c8;
        border-bottom: 1px solid rgba(228, 222, 210, 0.9);
        background: transparent;
        padding: 14px 12px 12px;
    }

    .answer-section-head {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 12px;
        border-bottom: 1px solid #e4ded2;
        padding-bottom: 10px;
    }

    .answer-section-actions {
        flex: 0 0 auto;
        display: flex;
        align-items: center;
        justify-content: flex-end;
        flex-wrap: wrap;
        gap: 8px;
        max-width: 45%;
    }

    .answer-section-list strong {
        display: block;
        color: #1c1b18;
        overflow-wrap: anywhere;
    }

    .source-title {
        display: block;
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

    .answer-section-head span {
        display: block;
        margin-top: 3px;
        color: #8a8678;
        font-size: 11px;
        text-transform: uppercase;
    }

    .answer-section a,
    .answer-section-head a,
    .answer-section-actions button {
        width: max-content;
        max-width: 100%;
        border-bottom: 1px solid #bdb7a8;
        border-top: 0;
        border-right: 0;
        border-left: 0;
        border-radius: 0;
        background: transparent;
        color: #1f5f8b;
        font-size: 12px;
        font-weight: 700;
        overflow-wrap: anywhere;
        text-decoration: none;
        padding: 0 0 2px;
    }

    .answer-section-actions button {
        color: #1c1b18;
    }

    .answer-section-actions button:disabled {
        cursor: default;
        opacity: 0.45;
    }

    .answer-section-copy {
        margin-top: 10px;
    }

    .answer-section-list {
        display: grid;
        gap: 7px;
        margin-top: 12px;
        border-top: 1px solid #e4ded2;
        padding-top: 10px;
    }

    .answer-link-list {
        display: grid;
        gap: 5px;
        min-width: 0;
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
        grid-template-columns: minmax(0, 1fr) 34px;
        align-items: end;
        gap: 8px;
        padding: 10px 0 0;
        border-top: 1px solid #d7d2c8;
    }

    .composer textarea {
        resize: none;
        min-width: 0;
        min-height: 42px;
        max-height: max(120px, min(50vh, calc(100dvh - 260px)));
        border: 0;
        border-bottom: 1px solid #bdb7a8;
        border-radius: 0;
        background: transparent;
        padding: 8px 0 6px;
        outline: none;
        line-height: 18px;
        overflow-y: hidden;
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
        width: 34px;
        min-width: 34px;
        height: 34px;
        border: 0;
        border-bottom: 1px solid #1c1b18;
        border-radius: 0;
        background: transparent;
        padding: 0;
        color: #1c1b18;
        font-size: 16px;
        font-weight: 700;
        line-height: 1;
    }

    .composer button:hover:not(:disabled) {
        background: rgba(28, 27, 24, 0.06);
    }

    .composer button:disabled {
        border-bottom-color: #d7d2c8;
        color: #aaa395;
        cursor: not-allowed;
    }

    .composer button.stop-icon {
        color: #9c3f2d;
        border-bottom-color: #9c3f2d;
    }

    small {
        color: #8a8678;
    }
</style>
