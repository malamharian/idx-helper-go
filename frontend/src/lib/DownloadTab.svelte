<script lang="ts">
  import {
    companies,
    downloadDir,
    addLog,
    type CompanyState,
    type Attachment,
  } from "./stores";
  import CompanyRow from "./CompanyRow.svelte";
  import {
    FetchReports,
    StartDownload,
    CancelAllDownloads,
    SetConcurrency,
    SelectDirectory,
  } from "../../wailsjs/go/main/App";

  const currentYear = new Date().getFullYear();
  const years = Array.from({ length: currentYear - 2014 }, (_, i) =>
    String(currentYear - i)
  );

  let year = String(currentYear);
  let period = "Tahunan";
  let fetching = false;

  let cbXlsx = true;
  let cbPdf = false;
  let cbZip = false;
  let cbAll = false;
  let regexFilter = "";

  let emitensText = "";

  let concurrency = 5;
  let showResults = false;

  function onAllChanged() {
    if (cbAll) {
      cbXlsx = cbPdf = cbZip = true;
    }
  }

  function onTypeChanged() {
    cbAll = cbXlsx && cbPdf && cbZip;
  }

  function getEmitensFilter(): Set<string> {
    const lines = emitensText
      .split("\n")
      .map((l) => l.trim().toUpperCase())
      .filter(Boolean);
    return new Set(lines);
  }

  function filterAttachments(attachments: Attachment[]): Attachment[] {
    if (cbAll) return [...attachments];

    const exts = new Set<string>();
    if (cbXlsx) exts.add(".xlsx");
    if (cbPdf) exts.add(".pdf");
    if (cbZip) exts.add(".zip");

    let regexPat: RegExp | null = null;
    const raw = regexFilter.trim();
    if (raw) {
      try {
        regexPat = new RegExp(raw, "i");
      } catch {
        // invalid regex, ignore
      }
    }

    return attachments.filter((att) => {
      if (exts.has(att.File_Type)) return true;
      if (regexPat && regexPat.test(att.File_Name)) return true;
      return false;
    });
  }

  function summarizeFilters(): string {
    if (cbAll) return "all file types";
    const parts: string[] = [];
    if (cbXlsx) parts.push(".xlsx");
    if (cbPdf) parts.push(".pdf");
    if (cbZip) parts.push(".zip");
    const raw = regexFilter.trim();
    if (raw) parts.push(`regex:${raw}`);
    return parts.join(", ") || "no file types selected";
  }

  async function onFetch() {
    if (!year.trim() || !period.trim()) {
      addLog("Tahun dan Periode harus diisi.");
      return;
    }

    CancelAllDownloads();
    fetching = true;

    try {
      const results = await FetchReports(year.trim(), period.trim());
      addLog(
        `API returned ${results?.length ?? 0} companies.`
      );

      if (!results || results.length === 0) {
        companies.set([]);
        showResults = false;
        fetching = false;
        return;
      }

      const emitens = getEmitensFilter();
      const newCompanies: CompanyState[] = [];

      for (const r of results) {
        const code = (r.code || "").toUpperCase();
        if (emitens.size > 0 && !emitens.has(code)) continue;
        const matched = filterAttachments(r.attachments || []);
        newCompanies.push({
          code,
          attachments: matched,
          status: "Ready",
          progress: 0,
          running: false,
        });
      }

      const emitensNote = emitens.size > 0 ? "filtered" : "all emitens";
      addLog(
        `Showing ${newCompanies.length} companies (${emitensNote}, ${summarizeFilters()}).`
      );
      companies.set(newCompanies);
      showResults = newCompanies.length > 0;
    } catch (err) {
      addLog(`Fetch error: ${err}`);
    }

    fetching = false;
  }

  async function onPickDir() {
    try {
      const dir = await SelectDirectory();
      if (dir) {
        downloadDir.set(dir);
      }
    } catch (err) {
      addLog(`Error selecting directory: ${err}`);
    }
  }

  function onStartAll() {
    if (!$downloadDir) {
      addLog("Please select a download directory first.");
      return;
    }
    for (const c of $companies) {
      if (!c.running && c.attachments.length > 0) {
        StartDownload(
          c.code,
          c.attachments,
          $downloadDir,
          year.trim(),
          period.trim().toLowerCase()
        );
      }
    }
  }

  function onCancelAll() {
    CancelAllDownloads();
  }

  function onConcurrencyChange() {
    const val = Math.max(1, Math.min(20, concurrency));
    SetConcurrency(val);
  }
</script>

<div class="flex flex-col gap-3 h-full">
  <!-- Controls -->
  <div class="flex flex-col gap-3">
    <!-- Year / Period / Fetch -->
    <div class="flex items-end gap-3">
      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-neutral-600 dark:text-neutral-400"
          >Tahun</span
        >
        <select
          bind:value={year}
          class="border border-neutral-300 dark:border-neutral-600 rounded-md px-2 py-1.5 text-sm bg-white dark:bg-neutral-700"
        >
          {#each years as y}
            <option value={y}>{y}</option>
          {/each}
        </select>
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-neutral-600 dark:text-neutral-400"
          >Periode</span
        >
        <select
          bind:value={period}
          class="border border-neutral-300 dark:border-neutral-600 rounded-md px-2 py-1.5 text-sm bg-white dark:bg-neutral-700"
        >
          <option>Tahunan</option>
          <option>TW1</option>
          <option>TW2</option>
          <option>TW3</option>
        </select>
      </label>
      <button
        on:click={onFetch}
        disabled={fetching}
        class="px-4 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
      >
        {#if fetching}
          <svg
            class="animate-spin h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
            ></path>
          </svg>
        {/if}
        Fetch Reports
      </button>
    </div>

    <!-- File types -->
    <div class="flex items-center gap-4 flex-wrap">
      <span class="text-xs font-medium text-neutral-600 dark:text-neutral-400"
        >File types:</span
      >
      <label class="flex items-center gap-1.5 text-sm">
        <input
          type="checkbox"
          bind:checked={cbXlsx}
          on:change={onTypeChanged}
          class="rounded"
        /> .xlsx
      </label>
      <label class="flex items-center gap-1.5 text-sm">
        <input
          type="checkbox"
          bind:checked={cbPdf}
          on:change={onTypeChanged}
          class="rounded"
        /> .pdf
      </label>
      <label class="flex items-center gap-1.5 text-sm">
        <input
          type="checkbox"
          bind:checked={cbZip}
          on:change={onTypeChanged}
          class="rounded"
        /> .zip
      </label>
      <label class="flex items-center gap-1.5 text-sm">
        <input
          type="checkbox"
          bind:checked={cbAll}
          on:change={onAllChanged}
          class="rounded"
        /> All
      </label>
      <input
        type="text"
        bind:value={regexFilter}
        placeholder="Custom regex e.g. Annual.*\.pdf"
        class="border border-neutral-300 dark:border-neutral-600 rounded-md px-2 py-1 text-sm w-52 bg-white dark:bg-neutral-700"
      />
    </div>

    <!-- Emitens -->
    <textarea
      bind:value={emitensText}
      placeholder="Kode Emiten (opsional) — one code per row. Leave empty for all."
      rows="4"
      class="border border-neutral-300 dark:border-neutral-600 rounded-md px-2 py-1.5 text-sm bg-white dark:bg-neutral-700 resize-y"
    ></textarea>
  </div>

  <!-- Results section -->
  {#if showResults}
    <hr class="border-neutral-200 dark:border-neutral-700" />

    <div class="flex items-center gap-3 flex-wrap">
      <button
        on:click={onPickDir}
        class="px-3 py-1.5 text-sm rounded-md bg-neutral-200 dark:bg-neutral-600 hover:bg-neutral-300 dark:hover:bg-neutral-500"
      >
        Choose Directory
      </button>
      <span
        class="text-sm truncate max-w-xs {$downloadDir
          ? 'text-neutral-800 dark:text-neutral-200'
          : 'italic text-neutral-400'}"
      >
        {$downloadDir || "No directory selected"}
      </span>
      <div class="flex-1"></div>
      <label class="flex items-center gap-1.5">
        <span class="text-xs text-neutral-500">Concurrency:</span>
        <select
          bind:value={concurrency}
          on:change={onConcurrencyChange}
          class="border border-neutral-300 dark:border-neutral-600 rounded-md px-2 py-1 text-sm bg-white dark:bg-neutral-700"
        >
          {#each [1, 2, 3, 5, 8, 10, 15, 20] as n}
            <option value={n}>{n}</option>
          {/each}
        </select>
      </label>
      <button
        on:click={onStartAll}
        class="px-3 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-700"
      >
        Start All
      </button>
      <button
        on:click={onCancelAll}
        class="px-3 py-1.5 text-sm rounded-md border border-neutral-300 dark:border-neutral-600 hover:bg-neutral-100 dark:hover:bg-neutral-700"
      >
        Cancel All
      </button>
    </div>

    <div
      class="flex-1 overflow-y-auto border border-neutral-300 dark:border-neutral-600 rounded-lg divide-y divide-neutral-200 dark:divide-neutral-700"
    >
      {#each $companies as company (company.code)}
        <CompanyRow
          {company}
          downloadDir={$downloadDir}
          {year}
          {period}
        />
      {/each}
    </div>
  {/if}
</div>
