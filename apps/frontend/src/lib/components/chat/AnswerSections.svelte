<script lang="ts">
    import type { AnswerSection } from "$lib/types";
    import type { EvidenceBasketItem } from "$lib/workflow/types";
    import SourceAnswerSection from "$lib/components/chat/SourceAnswerSection.svelte";

    export let sections: AnswerSection[] = [];
    export let messageID = "";
    export let basketItems: EvidenceBasketItem[] = [];
    export let onAskEvidence: (prompt: string) => void | Promise<void> = () => {};
    export let onPinEvidence: (item: EvidenceBasketItem) => void | Promise<void> = () => {};

    const visibleSectionLimit = 3;
    $: visibleSections = sections.slice(0, visibleSectionLimit);
    $: hiddenSections = sections.slice(visibleSectionLimit);
</script>

{#if sections.length}
    <div class="answer-sections" aria-label="Structured source answer">
        {#each visibleSections as section, index (`section-${messageID}-${index}`)}
            <SourceAnswerSection
                {section}
                {messageID}
                {basketItems}
                {onAskEvidence}
                {onPinEvidence}
            />
        {/each}
        {#if hiddenSections.length}
            <details class="more-sections">
                <summary>More source context ({hiddenSections.length})</summary>
                {#each hiddenSections as section, index (`hidden-section-${messageID}-${index}`)}
                    <SourceAnswerSection
                        {section}
                        {messageID}
                        {basketItems}
                        {onAskEvidence}
                        {onPinEvidence}
                        quiet
                    />
                {/each}
            </details>
        {/if}
    </div>
{/if}

<style>
    .answer-sections {
        display: grid;
        gap: 12px;
        margin-top: 14px;
    }

    .more-sections {
        border-top: 1px solid #d7d2c8;
        padding-top: 10px;
    }

    summary {
        cursor: pointer;
        color: #625f55;
        font-weight: 700;
    }
</style>
