<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let disabled = false;
    export let placeholder = "Ask anything or type 'help' for commands…";

    const dispatch = createEventDispatcher<{ send: string }>();

    let value = "";

    function handleKeyDown(e: KeyboardEvent) {
        if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            submit();
        }
    }

    function submit() {
        const text = value.trim();
        if (!text || disabled) return;
        dispatch("send", text);
        value = "";
    }
</script>

<div class="chat-input-row">
    <textarea
        bind:value
        {placeholder}
        {disabled}
        rows="1"
        class="input"
        on:keydown={handleKeyDown}
    ></textarea>
    <button class="send-btn" on:click={submit} {disabled}> ↑ </button>
</div>

<style>
    .chat-input-row {
        display: flex;
        align-items: flex-end;
        gap: 0.5rem;
        padding: 0.75rem 1rem;
        border-top: 1px solid #e5e7eb;
        background: #fff;
    }

    .input {
        flex: 1;
        resize: none;
        border: 1px solid #d1d5db;
        border-radius: 0.5rem;
        padding: 0.5rem 0.75rem;
        font-size: 0.9rem;
        font-family: inherit;
        line-height: 1.4;
        outline: none;
        transition: border-color 0.15s;
        max-height: 8rem;
        overflow-y: auto;
    }
    .input:focus {
        border-color: #2563eb;
    }

    .send-btn {
        width: 2.25rem;
        height: 2.25rem;
        border-radius: 50%;
        background: #2563eb;
        color: #fff;
        font-size: 1.1rem;
        border: none;
        cursor: pointer;
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: background 0.15s;
    }
    .send-btn:hover:not(:disabled) {
        background: #1d4ed8;
    }
    .send-btn:disabled {
        background: #9ca3af;
        cursor: not-allowed;
    }
</style>
