<script lang="ts">
  /** Visual variant. `ghost` is a small transparent button for inline actions. */
  export let variant: "primary" | "secondary" | "ghost" = "primary";
  /** When true, the button renders as disabled and slot content shows as-is. */
  export let loading = false;
  /** Additional disabled condition from the caller (e.g. empty URI). */
  export let disabled = false;
  export let type: "button" | "submit" = "button";
</script>

<button
  {type}
  class:connector-button={variant !== "ghost"}
  class:connector-button-secondary={variant === "secondary"}
  class:ghost={variant === "ghost"}
  disabled={disabled || loading}
  on:click
  {...$$restProps}
>
  <slot />
</button>

<style>
  .ghost {
    background: transparent;
    color: #374151;
    border: 1px solid #d1d5db;
    padding: 0.25rem 0.7rem;
    border-radius: 5px;
    font-size: 0.8rem;
    font-weight: normal;
    cursor: pointer;
  }
  .ghost:hover:not(:disabled) {
    background: #f3f4f6;
  }
  .ghost:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
