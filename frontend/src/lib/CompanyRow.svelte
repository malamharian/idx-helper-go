<script lang="ts">
  import type { CompanyState } from "./stores";
  import { StartDownload, CancelDownload } from "../../wailsjs/go/main/App";

  export let company: CompanyState;
  export let downloadDir: string;
  export let year: string;
  export let period: string;

  function handleToggle() {
    if (company.running) {
      CancelDownload(company.code);
    } else {
      StartDownload(
        company.code,
        company.attachments,
        downloadDir,
        year,
        period.toLowerCase()
      );
    }
  }
</script>

<div class="px-3 py-2 rounded-md hover:bg-neutral-100 dark:hover:bg-neutral-700/50">
  <div class="flex items-center gap-3">
    <span class="font-bold text-sm w-16 shrink-0">{company.code}</span>
    <span class="text-xs text-neutral-500 dark:text-neutral-400 w-20 shrink-0">
      {company.attachments.length} file(s)
    </span>
    <div class="w-32 shrink-0">
      <div class="w-full bg-neutral-200 dark:bg-neutral-600 rounded-full h-1.5">
        <div
          class="bg-blue-500 h-1.5 rounded-full transition-all duration-200"
          style="width: {company.progress * 100}%"
        ></div>
      </div>
    </div>
    <span class="text-xs w-36 shrink-0 truncate">{company.status}</span>
    <button
      on:click={handleToggle}
      class="text-xs px-3 py-1 rounded-md shrink-0 {company.running
        ? 'bg-red-100 text-red-700 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-400 dark:hover:bg-red-900/50'
        : 'bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:hover:bg-blue-900/50'}"
    >
      {company.running ? "Cancel" : "Start"}
    </button>
  </div>
  <div class="mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
    {#each company.attachments as att}
      <span class="text-[11px] text-neutral-400 dark:text-neutral-500">
        {att.File_Name}
      </span>
    {/each}
  </div>
</div>
