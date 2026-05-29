<script lang="ts">
  export let title: string;
  export let description: string;
  export let examples: string[] = [];
</script>

<section class="card">
  <h2>{title}</h2>
  <p class="hint">{description}</p>

  {#if examples.length > 0}
    <div class="examples" aria-label="URI examples">
      {#each examples as example}
        <code>{example}</code>
      {/each}
    </div>
  {/if}

  <slot />
</section>

<style>
  .card {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 1.25rem 1.5rem;
    margin-bottom: 1.5rem;
  }

  h2 {
    font-size: 0.875rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #6b7280;
    margin: 0 0 1rem;
  }

  .hint {
    color: #6b7280;
    font-size: 0.85rem;
    margin: 0 0 1rem;
  }

  .examples {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
    margin-bottom: 0.9rem;
  }

  .examples code {
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    color: #374151;
    font-size: 0.78rem;
    padding: 0.2rem 0.35rem;
    word-break: break-all;
  }

  :global(.connector-badge) {
    display: inline-block;
    margin-bottom: 0.75rem;
    padding: 0.3rem 0.75rem;
    background: #f0fdf4;
    color: #166534;
    border: 1px solid #bbf7d0;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 500;
  }

  :global(.connector-mode-toggle) {
    display: inline-flex;
    gap: 0.25rem;
    padding: 0.25rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    margin: 0 0 0.75rem;
    background: #f9fafb;
  }

  :global(.connector-mode-toggle button) {
    background: transparent;
    color: #374151;
    border: 0;
    padding: 0.4rem 0.65rem;
    border-radius: 4px;
    font-weight: normal;
  }

  :global(.connector-mode-toggle button.active) {
    background: #111827;
    color: white;
  }

  :global(.connector-field) {
    display: block;
    margin-bottom: 0.75rem;
    font-size: 0.85rem;
  }

  :global(.connector-field > span) {
    display: block;
    margin-bottom: 0.25rem;
    color: #374151;
  }

  :global(.connector-field-offset) {
    margin-top: 0.75rem;
  }

  :global(.connector-optional) {
    font-weight: 400;
    color: #9ca3af;
    font-size: 0.8rem;
  }

  :global(.connector-help) {
    margin: -0.25rem 0 0.75rem;
    font-size: 0.8rem;
    color: #6b7280;
  }

  :global(.connector-help summary) {
    cursor: pointer;
    color: #2563eb;
    user-select: none;
  }

  :global(.connector-help ol) {
    margin: 0.5rem 0 0.5rem 1.25rem;
    padding: 0;
    line-height: 1.8;
  }

  :global(.connector-note) {
    margin: 0.25rem 0 0;
    color: #9ca3af;
  }

  :global(.connector-input),
  :global(.connector-textarea) {
    width: 100%;
    padding: 0.5rem 0.6rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font: inherit;
    box-sizing: border-box;
  }

  :global(.connector-textarea) {
    min-height: 9rem;
    resize: vertical;
  }

  :global(.connector-table-wrap) {
    overflow-x: auto;
    margin: 0.25rem 0 1rem;
  }

  :global(.connector-format-table) {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.82rem;
  }

  :global(.connector-format-table th),
  :global(.connector-format-table td) {
    border: 1px solid #e5e7eb;
    padding: 0.45rem 0.55rem;
    text-align: left;
    vertical-align: top;
  }

  :global(.connector-format-table th) {
    background: #f9fafb;
    color: #374151;
    font-weight: 600;
  }

  :global(.connector-button) {
    background: #111827;
    color: white;
    border: 0;
    padding: 0.55rem 1rem;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
  }

  :global(.connector-button:disabled) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.connector-log) {
    margin-top: 0.75rem;
    padding: 0.6rem 0.8rem;
    background: #111827;
    color: #d1fae5;
    border-radius: 6px;
    font-size: 0.78rem;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 200px;
    overflow-y: auto;
  }

  :global(.connector-error) {
    margin-top: 1rem;
    padding: 0.75rem 1rem;
    background: #fef2f2;
    color: #991b1b;
    border: 1px solid #fecaca;
    border-radius: 6px;
    font-size: 0.85rem;
    white-space: pre-wrap;
  }

  :global(.connector-card-code) {
    background: #f3f4f6;
    padding: 0.1rem 0.3rem;
    border-radius: 4px;
  }
</style>
