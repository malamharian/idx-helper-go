<script lang="ts">
  import { aggInputDir, aggOutputPath, aggStatus, addLog } from "./stores";
  import {
    StartAggregate,
    CancelAggregate,
    SelectDirectory,
    SelectSaveFile,
  } from "../../wailsjs/go/main/App";

  $: isRunning = $aggStatus.status === "running";

  async function onPickInput() {
    try {
      const dir = await SelectDirectory();
      if (dir) aggInputDir.set(dir);
    } catch (err) {
      addLog(`Error selecting directory: ${err}`);
    }
  }

  async function onPickOutput() {
    try {
      const path = await SelectSaveFile();
      if (path) {
        aggOutputPath.set(path.endsWith(".xlsx") ? path : path + ".xlsx");
      }
    } catch (err) {
      addLog(`Error selecting file: ${err}`);
    }
  }

  function onAggregate() {
    if (!$aggInputDir) {
      addLog("Please select an input directory.");
      return;
    }
    if (!$aggOutputPath) {
      addLog("Please select an output file.");
      return;
    }
    StartAggregate($aggInputDir, $aggOutputPath);
  }

  function onCancel() {
    CancelAggregate();
  }
</script>

<div class="flex flex-col gap-4">
  <p class="text-sm text-neutral-500 dark:text-neutral-400">
    Merge all .xlsx files from a directory into a single file, grouped by sheet
    name.
  </p>

  <div class="flex items-center gap-3">
    <button
      on:click={onPickInput}
      class="px-3 py-1.5 text-sm rounded-md bg-neutral-200 dark:bg-neutral-600 hover:bg-neutral-300 dark:hover:bg-neutral-500 shrink-0"
    >
      Input Directory
    </button>
    <span
      class="text-sm truncate {$aggInputDir
        ? 'text-neutral-800 dark:text-neutral-200'
        : 'italic text-neutral-400'}"
    >
      {$aggInputDir || "No directory selected"}
    </span>
  </div>

  <div class="flex items-center gap-3">
    <button
      on:click={onPickOutput}
      class="px-3 py-1.5 text-sm rounded-md bg-neutral-200 dark:bg-neutral-600 hover:bg-neutral-300 dark:hover:bg-neutral-500 shrink-0"
    >
      Output File
    </button>
    <span
      class="text-sm truncate {$aggOutputPath
        ? 'text-neutral-800 dark:text-neutral-200'
        : 'italic text-neutral-400'}"
    >
      {$aggOutputPath || "No output file selected"}
    </span>
  </div>

  <div class="flex items-center gap-3">
    <button
      on:click={onAggregate}
      disabled={isRunning}
      class="px-4 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
    >
      Aggregate
    </button>
    <button
      on:click={onCancel}
      disabled={!isRunning}
      class="px-4 py-1.5 text-sm rounded-md border border-neutral-300 dark:border-neutral-600 hover:bg-neutral-100 dark:hover:bg-neutral-700 disabled:opacity-50 disabled:cursor-not-allowed"
    >
      Cancel
    </button>
    <span class="text-sm text-neutral-600 dark:text-neutral-400">
      {$aggStatus.message}
    </span>
  </div>

  {#if isRunning}
    <div class="w-full bg-neutral-200 dark:bg-neutral-600 rounded-full h-1.5">
      <div
        class="bg-blue-500 h-1.5 rounded-full animate-pulse w-full"
      ></div>
    </div>
  {/if}
</div>
