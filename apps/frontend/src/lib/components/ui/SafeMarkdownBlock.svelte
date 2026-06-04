<script lang="ts">
    import InlineText from "$lib/components/ui/InlineText.svelte";
    import { messageLines } from "$lib/findings/viewModel";

    export let text = "";
    export let emptyText = "";
    export let variant: "message" | "plain" | "source" | "detail" = "message";

    $: displayText = text.trim() || emptyText;
    $: lines = messageLines(displayText);
</script>

<div
    class="safe-markdown"
    class:message={variant === "message"}
    class:plain={variant === "plain"}
    class:source={variant === "source"}
    class:detail={variant === "detail"}
>
    {#each lines as line}
        {#if line.kind === "blank"}
            <div class="message-gap"></div>
        {:else if line.kind === "heading"}
            <h4><InlineText text={line.text} /></h4>
        {:else if line.kind === "section"}
            <h4 class="section-line"><InlineText text={line.text} /></h4>
        {:else if line.kind === "number"}
            <p class="number-line"><InlineText text={line.text} /></p>
        {:else if line.kind === "bullet"}
            <p class="bullet-line"><InlineText text={line.text} /></p>
        {:else}
            <p class="body-line"><InlineText text={line.text} /></p>
        {/if}
    {/each}
</div>

<style>
    .safe-markdown {
        min-width: 0;
        display: grid;
        gap: 7px;
        color: inherit;
        line-height: 1.5;
    }

    .source,
    .detail {
        gap: 6px;
        color: #5f5b50;
        line-height: 1.55;
    }

    p,
    h4 {
        min-width: 0;
        margin: 0;
        overflow-wrap: anywhere;
    }

    h4 {
        margin-top: 10px;
        color: #28261f;
        font-size: 13px;
        line-height: 1.35;
    }

    h4:first-child {
        margin-top: 0;
    }

    .section-line {
        display: flex;
        align-items: center;
        gap: 8px;
        margin: 14px 0 3px;
        border-top: 1px solid #d7d2c8;
        border-bottom: 1px solid #e4ded2;
        padding: 9px 10px 8px;
        color: #1c1b18;
        letter-spacing: 0.04em;
        text-transform: uppercase;
    }

    .section-line:first-child {
        margin-top: 0;
    }

    .section-line::before {
        content: "";
        width: 7px;
        height: 7px;
        flex: 0 0 auto;
        border-radius: 50%;
        background: #8a6a20;
    }

    .message .body-line {
        border-left: 2px solid #d7d2c8;
        padding: 3px 0 3px 12px;
    }

    .message .body-line + .body-line {
        border-top: 1px solid rgba(215, 210, 200, 0.48);
        padding-top: 8px;
    }

    .number-line {
        margin-top: 5px;
        color: #1c1b18;
        font-weight: 700;
    }

    .bullet-line {
        position: relative;
        padding-left: 24px;
    }

    .bullet-line::before {
        content: "";
        position: absolute;
        left: 12px;
        top: 0.72em;
        width: 5px;
        height: 5px;
        border-radius: 50%;
        background: #8a8678;
    }

    .message-gap {
        height: 4px;
    }

    .detail {
        font-size: 12px;
    }

    .detail .section-line,
    .source .section-line {
        margin-top: 10px;
    }
</style>
