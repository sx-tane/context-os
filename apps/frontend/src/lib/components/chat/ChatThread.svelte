<script lang="ts">
    import { afterUpdate } from "svelte";
    import type { ChatMessage } from "$lib/types";
    import ChatMessageComponent from "./ChatMessage.svelte";

    export let messages: ChatMessage[];

    let container: HTMLDivElement;

    afterUpdate(() => {
        if (container) {
            container.scrollTop = container.scrollHeight;
        }
    });
</script>

<div class="thread" bind:this={container}>
    {#each messages as msg (msg.id)}
        <ChatMessageComponent message={msg} />
    {/each}
</div>

<style>
    .thread {
        flex: 1;
        overflow-y: auto;
        padding: 1rem 1.25rem;
        display: flex;
        flex-direction: column;
    }
</style>
