<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
  import {
    activeTab,
    addLog,
    updateCompanyProgress,
    aggStatus,
    type DownloadProgress,
    type AggregateProgress,
  } from "./lib/stores";
  import DownloadTab from "./lib/DownloadTab.svelte";
  import AggregateTab from "./lib/AggregateTab.svelte";
  import LogPanel from "./lib/LogPanel.svelte";

  onMount(() => {
    EventsOn("log", (msg: string) => addLog(msg));
    EventsOn("download:progress", (p: DownloadProgress) =>
      updateCompanyProgress(p)
    );
    EventsOn("aggregate:progress", (p: AggregateProgress) =>
      aggStatus.set(p)
    );
  });

  onDestroy(() => {
    EventsOff("log");
    EventsOff("download:progress");
    EventsOff("aggregate:progress");
  });
</script>

<div class="flex flex-col h-screen bg-white dark:bg-neutral-800 text-neutral-900 dark:text-neutral-100">
  <!-- Tab bar -->
  <div class="flex border-b border-neutral-200 dark:border-neutral-700 shrink-0">
    <button
      on:click={() => activeTab.set("download")}
      class="px-6 py-2.5 text-sm font-medium transition-colors
        {$activeTab === 'download'
          ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400'
          : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-300'}"
    >
      Download
    </button>
    <button
      on:click={() => activeTab.set("aggregate")}
      class="px-6 py-2.5 text-sm font-medium transition-colors
        {$activeTab === 'aggregate'
          ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400'
          : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-300'}"
    >
      Aggregate
    </button>
  </div>

  <!-- Tab content — both tabs stay mounted to preserve state -->
  <div class="flex-1 overflow-hidden relative">
    <div class="absolute inset-0 p-4 overflow-auto" class:hidden={$activeTab !== 'download'}>
      <DownloadTab />
    </div>
    <div class="absolute inset-0 p-4 overflow-auto" class:hidden={$activeTab !== 'aggregate'}>
      <AggregateTab />
    </div>
  </div>

  <!-- Log panel -->
  <div class="shrink-0 px-4 pb-4">
    <LogPanel />
  </div>
</div>
