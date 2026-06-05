<script lang="ts">
    import type { AnswerSection } from "$lib/types";
    import type { EvidenceBasketItem } from "$lib/workflow/types";
    import SafeMarkdownBlock from "$lib/components/ui/SafeMarkdownBlock.svelte";
    import { markdownBulletList } from "$lib/findings/viewModel";
    import {
        askChatPromptForEvidence,
        basketItemFromAnswerSection,
    } from "$lib/workflow/viewModel";

    export let section: AnswerSection;
    export let messageID = "";
    export let basketItems: EvidenceBasketItem[] = [];
    export let quiet = false;
    export let onAskEvidence: (prompt: string) => void | Promise<void> = () => {};
    export let onPinEvidence: (item: EvidenceBasketItem) => void | Promise<void> = () => {};

    const visibleItemLimit = 3;

    $: link = sectionLink(section);
    $: basketItem = basketItemFromAnswerSection(section, messageID);

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

    function visibleItems(items: string[] | undefined) {
        return (items ?? []).slice(0, visibleItemLimit);
    }

    function hiddenItems(items: string[] | undefined) {
        return (items ?? []).slice(visibleItemLimit);
    }

    function hiddenItemCount(items: string[] | undefined) {
        return Math.max(0, (items?.length ?? 0) - visibleItemLimit);
    }
</script>

<section class={`answer-section ${connectorClass(section.connector)}`} class:quiet>
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
                text={markdownBulletList(visibleItems(section.facts))}
                variant="source"
            />
            {#if hiddenItemCount(section.facts)}
                <details class="more-items">
                    <summary>{hiddenItemCount(section.facts)} more facts</summary>
                    <SafeMarkdownBlock
                        text={markdownBulletList(hiddenItems(section.facts))}
                        variant="source"
                    />
                </details>
            {/if}
        </div>
    {/if}
    {#if section.open_items?.length}
        <div class="answer-section-list">
            <strong>Open items</strong>
            <SafeMarkdownBlock
                text={markdownBulletList(visibleItems(section.open_items))}
                variant="source"
            />
            {#if hiddenItemCount(section.open_items)}
                <details class="more-items">
                    <summary>{hiddenItemCount(section.open_items)} more open items</summary>
                    <SafeMarkdownBlock
                        text={markdownBulletList(hiddenItems(section.open_items))}
                        variant="source"
                    />
                </details>
            {/if}
        </div>
    {/if}
    {#if section.coding_notes?.length}
        <div class="answer-section-list">
            <strong>Coding notes</strong>
            <SafeMarkdownBlock
                text={markdownBulletList(visibleItems(section.coding_notes))}
                variant="source"
            />
            {#if hiddenItemCount(section.coding_notes)}
                <details class="more-items">
                    <summary>{hiddenItemCount(section.coding_notes)} more coding notes</summary>
                    <SafeMarkdownBlock
                        text={markdownBulletList(hiddenItems(section.coding_notes))}
                        variant="source"
                    />
                </details>
            {/if}
        </div>
    {/if}
    {#if section.links?.length}
        <div class="answer-section-list">
            <strong>Links</strong>
            {#if sectionURLLinks(section).length}
                <div class="answer-link-list">
                    {#each visibleItems(sectionURLLinks(section)) as item}
                        <a href={item} target="_blank" rel="noreferrer">{item}</a>
                    {/each}
                </div>
            {/if}
            {#if sectionTextLinks(section).length}
                <SafeMarkdownBlock
                    text={markdownBulletList(visibleItems(sectionTextLinks(section)))}
                    variant="source"
                />
            {/if}
            {#if hiddenItemCount(section.links)}
                <details class="more-items">
                    <summary>{hiddenItemCount(section.links)} more links</summary>
                    <SafeMarkdownBlock
                        text={markdownBulletList(hiddenItems(section.links))}
                        variant="source"
                    />
                </details>
            {/if}
        </div>
    {/if}
</section>

<style>
    .answer-section {
        border-top: 1px solid #d7d2c8;
        border-bottom: 1px solid rgba(228, 222, 210, 0.9);
        background: transparent;
        padding: 14px 12px 12px;
    }

    .answer-section.quiet {
        padding-left: 0;
        padding-right: 0;
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

    .more-items {
        margin-top: 4px;
        border-top: 1px solid #e4ded2;
        padding-top: 6px;
    }

    summary {
        cursor: pointer;
        color: #625f55;
        font-weight: 700;
    }
</style>
