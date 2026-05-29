<script lang="ts">
  /** Log text to display. */
  export let log = "";
  /** Whether an operation is currently running (shows placeholder when log is empty). */
  export let loading = false;
  /** When false the panel is hidden entirely (e.g. when provider !== "codex"). */
  export let visible = true;
  export let placeholder = "Waiting for output\u2026";
</script>

{#if visible && (log || loading)}
  <pre class="connector-log">{log || placeholder}</pre>
{/if}
