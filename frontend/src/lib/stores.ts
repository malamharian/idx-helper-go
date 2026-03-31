import { writable } from "svelte/store";
import type { main } from "../../wailsjs/go/models";

export type Attachment = main.Attachment;
export type CompanyResult = main.CompanyResult;

export interface CompanyState {
  code: string;
  attachments: Attachment[];
  status: string;
  progress: number;
  running: boolean;
}

export interface DownloadProgress {
  code: string;
  status?: string;
  progress?: number;
  running?: boolean;
}

export interface AggregateProgress {
  status: string;
  message: string;
}

export const activeTab = writable<"download" | "aggregate">("download");
export const logs = writable<string[]>([]);
export const companies = writable<CompanyState[]>([]);
export const downloadDir = writable<string>("");
export const aggInputDir = writable<string>("");
export const aggOutputPath = writable<string>("");
export const aggStatus = writable<AggregateProgress>({
  status: "idle",
  message: "",
});

const MAX_LOGS = 500;

export function addLog(msg: string) {
  logs.update((l) => {
    const next = [...l, msg];
    if (next.length > MAX_LOGS) {
      next.splice(0, next.length - MAX_LOGS);
    }
    return next;
  });
}

export function updateCompanyProgress(p: DownloadProgress) {
  companies.update((list) =>
    list.map((c) => {
      if (c.code !== p.code) return c;
      return {
        ...c,
        ...(p.status !== undefined && { status: p.status }),
        ...(p.progress !== undefined && { progress: p.progress }),
        ...(p.running !== undefined && { running: p.running }),
      };
    })
  );
}
