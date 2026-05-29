<script lang="ts">
  /** Currently selected value — use bind:value for two-way sync. */
  export let value: string;
  /** Tab options rendered as toggle buttons. */
  export let options: { value: string; label: string }[];
  export let ariaLabel = "";
</script>

<div class="connector-mode-toggle" aria-label={ariaLabel}>
  {#each options as opt}
    <button
      type="button"
      class:active={value === opt.value}
      on:click={() => (value = opt.value)}
    >
      {opt.label}
    </button>
  {/each}
</div>
