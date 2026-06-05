<script lang="ts">
  import { tick } from "svelte";
  import { composerHeight } from "$lib/chat/composer";
  import { isNearBottom, localDBStatusLine } from "$lib/chat/controller";
  import type { ChatMessage, ChatQueryMode, ChatQueryResult } from "$lib/types";
  import type { EvidenceBasketItem } from "$lib/workflow/types";
  import AnswerSections from "$lib/components/chat/AnswerSections.svelte";
  import ChatStreamPanel from "$lib/components/chat/ChatStreamPanel.svelte";
  import SafeMarkdownBlock from "$lib/components/ui/SafeMarkdownBlock.svelte";
  import {
    artifactLink,
    artifactSourceLabel,
    findingDescription,
    findingImpact,
    findingRecommendedAction,
    findingSummary,
    formatTime,
    reviewCandidateCount,
    severityLabel,
    topActionableFindings,
  } from "$lib/findings/viewModel";

  export let messages: ChatMessage[] = [];
  export let hasSources = false;
  export let busy = false;
  export let canStop = false;
  export let mode: ChatQueryMode = "auto";
  export let command = "";
  export let basketItems: EvidenceBasketItem[] = [];
  export let onClear: () => void | Promise<void> = () => {};
  export let onSubmit: () => void | Promise<void> = () => {};
  export let onStop: () => void | Promise<void> = () => {};
  export let onModeChange: (
    mode: ChatQueryMode,
  ) => void | Promise<void> = () => {};
  export let onAskEvidence: (prompt: string) => void | Promise<void> = () => {};
  export let onPinEvidence: (
    item: EvidenceBasketItem,
  ) => void | Promise<void> = () => {};

  let messagesEl: HTMLDivElement | null = null;
  let composerForm: HTMLFormElement | null = null;
  let composerTextarea: HTMLTextAreaElement | null = null;
  let stickToBottom = true;
  let expandedStreams: Record<string, boolean> = {};
  let lastMessageSignature = "";

  $: messageSignature = messages
    .map((message) =>
      [
        message.id,
        message.text,
        message.loading ? "loading" : "done",
        message.stream?.status ?? "",
        message.stream?.latestLine ?? "",
        message.stream?.lines.length ?? 0,
      ].join(":"),
    )
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
    const pieces = [`Provider: ${queryProviderLabel(result.provider)}`];
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
    pieces.push(
      `Artifacts: ${result.artifact_count ?? result.artifacts?.length ?? 0}`,
    );
    return pieces.join(" · ");
  }

  function sourceTraceLabel(result: ChatQueryResult) {
    if (result.artifacts?.length) return "Source trace";
    if (result.provider === "codex") return "Live source trace";
    return "Source trace";
  }

  function answerSections(result?: ChatQueryResult) {
    return (
      result?.answer_sections?.filter((section) =>
        Boolean(
          section.source_label ||
            section.source_uri ||
            section.summary ||
            section.facts?.length ||
            section.open_items?.length ||
            section.coding_notes?.length ||
            section.links?.length,
        ),
      ) ?? []
    );
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
    const maxHeight =
      parseFloat(styles.maxHeight) || composerTextarea.scrollHeight;
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

  function setMode(nextMode: ChatQueryMode) {
    void onModeChange(nextMode);
  }
</script>

<section class="chat-card">
  <div class="chat-head">
    <div>
      <strong>Report Agent</strong>
      <span
        >{hasSources
          ? "Ask against selected sources"
          : "Connect sources before asking"}</span
      >
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
        <p>
          {hasSources
            ? "Ask about Slack messages, GitHub PRs, Jira tickets, docs, findings, or recent activity."
            : "Connect GitHub repos, Slack channels, or docs first."}
        </p>
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
              text={message.text ||
                (message.loading && !message.stream ? "Working..." : "")}
              variant={message.role === "user" ? "plain" : "message"}
            />
          </div>
          {#if message.stream}
            <ChatStreamPanel
              stream={message.stream}
              expanded={streamExpanded(message)}
              onToggle={() => toggleStream(message)}
            />
          {/if}
          {#if message.card?.kind === "query" && message.card.chatResult}
            {#if answerSections(message.card.chatResult).length}
              <AnswerSections
                sections={answerSections(message.card.chatResult)}
                messageID={message.id}
                {basketItems}
                {onAskEvidence}
                {onPinEvidence}
              />
            {/if}
            <div
              class="query-meta"
              class:live={message.card.chatResult.provider === "codex"}
            >
              <span>{queryProviderLabel(message.card.chatResult.provider)}</span
              >
              {#if querySourceLabel(message.card.chatResult)}
                <small>{querySourceLabel(message.card.chatResult)}</small>
              {/if}
            </div>
            {#if localDBStatusLine(message.card.chatResult)}
              <p
                class="local-db-status"
                class:saved={message.card.chatResult.evidence_save_status ===
                  "saved"}
                class:error={message.card.chatResult.evidence_save_status ===
                  "error"}
              >
                {localDBStatusLine(message.card.chatResult)}
              </p>
            {/if}
            <div
              class="source-trace"
              class:live={message.card.chatResult.provider === "codex" &&
                !message.card.chatResult.artifacts?.length}
            >
              <strong>{sourceTraceLabel(message.card.chatResult)}</strong>
              <span>{traceSummary(message.card.chatResult, message)}</span>
            </div>
          {/if}
          {#if message.card?.chatResult?.artifacts?.length}
            <details>
              <summary
                >{message.card.chatResult.artifact_count} evidence items</summary
              >
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
                      <a href={link} target="_blank" rel="noreferrer"
                        >Open source</a
                      >
                    {:else}
                      <span>{artifact.source_uri || "Stored local source"}</span
                      >
                    {/if}
                  </div>
                </div>
              {/each}
            </details>
          {/if}
          {#if message.card?.findingsResult}
            {@const topFindings = topActionableFindings(
              message.card.findingsResult,
              3,
            )}
            <details>
              <summary>
                {message.card.findingsResult.mismatch_count ??
                  topFindings.length} top issues · {reviewCandidateCount(
                  message.card.findingsResult,
                )} review candidates
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
                      <p>
                        <b>Recommended:</b>
                        {findingRecommendedAction(mismatch)}
                      </p>
                    {/if}
                  </div>
                {/each}
              {:else}
                <div class="evidence-item">
                  <p>
                    No top actionable issues. Dependency-only signals are under
                    Review candidates.
                  </p>
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
    <div class="composer-mode" aria-label="Chat source mode">
      {#each ["auto", "codex", "local"] as item}
        <button
          type="button"
          class:active-mode={mode === item}
          aria-pressed={mode === item}
          on:click={() => setMode(item as ChatQueryMode)}
          >{item === "auto"
            ? "Auto"
            : item === "codex"
              ? "Codex"
              : "Local"}</button
        >
      {/each}
    </div>
    <textarea
      bind:this={composerTextarea}
      bind:value={command}
      disabled={busy || !hasSources}
      placeholder={hasSources
        ? "Ask about PRs, Slack threads, findings, or recent activity..."
        : "Connect sources first..."}
      rows="1"
      on:input={resizeComposer}
      on:keydown={handleComposerKeydown}
    ></textarea>
    {#if canStop}
      <button
        type="button"
        class="send-icon stop-icon"
        aria-label="Stop Codex chat"
        title="Stop Codex chat"
        on:click={onStop}>×</button
      >
    {:else}
      <button
        class="send-icon"
        aria-label="Send message"
        title="Send"
        disabled={busy || !hasSources || command.trim() === ""}>↑</button
      >
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
    background-image: linear-gradient(
      90deg,
      #1c1b18 0 50%,
      transparent 50% 100%
    );
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
    background-image: linear-gradient(
      90deg,
      #b4422a 0 50%,
      transparent 50% 100%
    );
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
    grid-template-rows: auto auto;
    align-items: end;
    gap: 8px;
    padding: 10px 0 0;
    border-top: 1px solid #d7d2c8;
  }

    .composer-mode {
        grid-column: 1 / -1;
        display: flex;
        align-items: center;
        gap: 6px;
        min-width: 0;
    }

  .composer-mode button {
    width: auto;
    min-width: 0;
    height: 24px;
    flex: 0 0 auto;
    border-bottom-color: #d7d2c8;
    padding: 0 8px;
    color: #8a8678;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .composer-mode button.active-mode {
    border-bottom-color: #1c1b18;
    color: #1c1b18;
  }

  .composer textarea {
    box-sizing: border-box;
    resize: none;
    min-width: 0;
    min-height: 34px;
    max-height: max(120px, min(50vh, calc(100dvh - 260px)));
    border: 0;
    border-bottom: 1px solid #bdb7a8;
    border-radius: 0;
    background: transparent;
    padding: 7px 0 6px;
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
    flex: 0 0 34px;
    height: 34px;
    align-self: end;
    border: 0;
    border-bottom: 1px solid #1c1b18;
    border-radius: 0;
    background-color: transparent;
    background-image: linear-gradient(
      90deg,
      #1c1b18 0 50%,
      transparent 50% 100%
    );
    background-position: 100% 0;
    background-size: 200% 100%;
    padding: 0;
    color: #1c1b18;
    font-size: 16px;
    font-weight: 700;
    line-height: 1;
    transition:
      background-position 0.18s ease,
      color 0.15s,
      border-color 0.15s,
      opacity 0.15s;
  }

  .composer button:hover:not(:disabled) {
    border-bottom-color: #1c1b18;
    background-position: 0 0;
    color: #f8f6ef;
  }

  .composer button:disabled {
    border-bottom-color: #d7d2c8;
    background-image: none;
    color: #aaa395;
    cursor: not-allowed;
  }

  .composer button.stop-icon {
    background-image: linear-gradient(
      90deg,
      #9c3f2d 0 50%,
      transparent 50% 100%
    );
    color: #9c3f2d;
    border-bottom-color: #9c3f2d;
  }

  .composer button.stop-icon:hover:not(:disabled) {
    border-bottom-color: #9c3f2d;
    background-position: 0 0;
    color: #f8f6ef;
  }

  .composer .composer-mode button {
    width: auto;
    min-width: 0;
    height: 24px;
    flex: 0 0 auto;
    border-bottom-color: #d7d2c8;
    padding: 0 8px;
    color: #8a8678;
    font-size: 11px;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .composer .composer-mode button.active-mode {
    border-bottom-color: #1c1b18;
    background-position: 0 0;
    color: #f8f6ef;
  }

  small {
    color: #8a8678;
  }
</style>
