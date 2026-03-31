<script lang="ts">
  import { logs } from "./stores";
  import { afterUpdate } from "svelte";

  let container: HTMLDivElement;

  afterUpdate(() => {
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  });
</script>

{#if $logs.length > 0}
  <div
    bind:this={container}
    class="border border-neutral-300 dark:border-neutral-600 rounded-lg p-2 h-28 overflow-y-auto text-xs font-mono bg-neutral-50 dark:bg-neutral-800"
  >
    {#each $logs as msg}
      <div class="py-px select-text text-neutral-700 dark:text-neutral-300 break-all">
        {msg}
      </div>
    {/each}
  </div>
{/if}
