<script lang="ts">
  /** Error message to display. Renders nothing when empty. */
  export let message = "";
</script>

{#if message}
  <div class="connector-error">{message}</div>
{/if}
