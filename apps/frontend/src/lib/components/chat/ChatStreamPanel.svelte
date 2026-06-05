<script lang="ts">
    import type { ChatStreamState, ChatStreamStatus } from "$lib/types";

    export let stream: ChatStreamState;
    export let expanded = false;
    export let onToggle: () => void = () => {};

    const expandedStreamLineLimit = 80;

    function streamStatusLabel(status: ChatStreamStatus) {
        if (status === "complete") return "Complete";
        if (status === "error") return "Error";
        return "Running";
    }

    function visibleStreamLines() {
        return stream.lines.slice(-expandedStreamLineLimit);
    }

    function hiddenStreamLineCount() {
        return Math.max(0, stream.lines.length - expandedStreamLineLimit);
    }
</script>

<section
    class="stream-panel"
    class:running={stream.status === "running"}
    class:error={stream.status === "error"}
    aria-label="Live stream progress"
>
    <button
        type="button"
        class="stream-toggle"
        aria-expanded={expanded}
        on:click={onToggle}
    >
        <span>{streamStatusLabel(stream.status)}</span>
        <strong>{expanded ? "Hide stream" : "Show stream"}</strong>
    </button>
    {#if !expanded}
        <p class="stream-latest">{stream.latestLine}</p>
        {#if stream.summary}
            <p class="stream-summary">{stream.summary}</p>
        {/if}
    {:else}
        <div class="stream-lines">
            {#if hiddenStreamLineCount() > 0}
                <small>{hiddenStreamLineCount()} earlier stream lines hidden</small>
            {/if}
            {#each visibleStreamLines() as line, index (`stream-line-${index}`)}
                <p>{line}</p>
            {/each}
        </div>
    {/if}
</section>

<style>
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
        margin: 7px 0 0;
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
        margin: 0;
        color: #625f55;
        overflow-wrap: anywhere;
    }
</style>
