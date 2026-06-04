<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let eyebrow = "CONFIRM";
    export let title = "";
    export let description = "";
    export let detail = "";
    export let confirmLabel = "Confirm";
    export let busyLabel = "Working";
    export let cancelLabel = "Cancel";
    export let busy = false;

    const dispatch = createEventDispatcher<{
        cancel: void;
        confirm: void;
    }>();
</script>

<div class="modal-backdrop">
    <div
        class="confirm-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-modal-title"
        tabindex="-1"
    >
        <span>{eyebrow}</span>
        <h2 id="confirm-modal-title">{title}</h2>
        <p>
            {description}
            {#if detail}
                <strong>{detail}</strong>
            {/if}
        </p>
        <div class="modal-actions">
            <button type="button" on:click={() => dispatch("cancel")} disabled={busy}>
                {cancelLabel}
            </button>
            <button class="danger" type="button" on:click={() => dispatch("confirm")} disabled={busy}>
                {busy ? busyLabel : confirmLabel}
            </button>
        </div>
    </div>
</div>

<style>
    .modal-backdrop {
        position: fixed;
        inset: 0;
        z-index: 130;
        display: grid;
        place-items: center;
        background: rgba(28, 27, 24, 0.34);
        padding: 18px;
    }

    .confirm-modal {
        width: min(520px, 100%);
        border: 1px solid #1c1b18;
        background: #ebe8e0;
        box-shadow: 0 18px 48px rgba(28, 27, 24, 0.22);
        padding: 20px;
    }

    .confirm-modal > span {
        display: block;
        margin-bottom: 10px;
        color: #d85d3f;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.05em;
    }

    .confirm-modal h2 {
        margin: 0 0 12px;
        color: #1c1b18;
        font-size: 18px;
        line-height: 1.25;
    }

    .confirm-modal p {
        margin: 0;
        color: #625f55;
        font-size: 13px;
        line-height: 1.6;
    }

    .confirm-modal p strong {
        color: #1c1b18;
        overflow-wrap: anywhere;
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 22px;
        border-top: 1px solid #d7d2c8;
        padding: 14px 0 4px;
    }

    .modal-actions button {
        min-width: 92px;
        height: 36px;
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

    .modal-actions button:hover:not(:disabled) {
        border-bottom-color: #1c1b18;
        background-position: 0 0;
        color: #f8f6ef;
    }

    .modal-actions button.danger {
        background-image: linear-gradient(90deg, #d85d3f 0 50%, transparent 50% 100%);
        border-bottom-color: #d85d3f;
        color: #d85d3f;
    }

    .modal-actions button.danger:hover:not(:disabled) {
        background-position: 0 0;
        color: #fffdf7;
    }

    .modal-actions button:disabled {
        cursor: not-allowed;
        opacity: 0.45;
    }
</style>
