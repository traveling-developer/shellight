<script lang="ts">
  // A sandbox's phase state as shape and colour: filled circle = ok,
  // dashed ring = busy, triangle = error.
  let { state }: { state: string } = $props();
</script>

<svg class={state} width="10" height="10" viewBox="0 0 16 16" aria-hidden="true">
  {#if state === "error"}
    <path d="M8 1.8L14.6 13.6H1.4Z" />
  {:else if state === "busy"}
    <circle cx="8" cy="8" r="5.5" />
  {:else}
    <circle cx="8" cy="8" r="6" />
  {/if}
</svg>

<style>
  svg {
    flex: none;
    fill: var(--ok);
  }

  .busy {
    fill: none;
    stroke: var(--busy);
    stroke-width: 2.4;
    stroke-dasharray: 4 2.6;
  }

  .error {
    fill: var(--error);
  }
</style>
