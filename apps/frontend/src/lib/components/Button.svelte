<script lang="ts">
  /** Primary or secondary visual variant. */
  export let variant: "primary" | "secondary" = "primary";
  /** When true, the button renders as disabled and any slot content shows as-is. */
  export let loading = false;
  /** Additional disabled condition from the caller (e.g. empty URI). */
  export let disabled = false;
  export let type: "button" | "submit" = "button";
</script>

<button
  {type}
  class="connector-button"
  class:connector-button-secondary={variant === "secondary"}
  disabled={disabled || loading}
  on:click
>
  <slot />
</button>
