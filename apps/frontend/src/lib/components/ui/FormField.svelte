<script lang="ts">
  /** Visible label text for the field. */
  export let label: string;
  /** Two-way bound field value. */
  export let value = "";
  export let type = "text";
  export let placeholder = "";
  /** Short parenthetical note rendered next to the label in muted style. */
  export let optional = "";
  /** Adds top margin (connector-field-offset) when the field follows a conditional block. */
  export let offset = false;
</script>

<label class="connector-field" class:connector-field-offset={offset}>
  <span>
    {label}{#if optional}<span class="connector-optional">
        {optional}</span
      >{/if}
  </span>
  <input class="connector-input" {type} bind:value {placeholder} />
</label>
