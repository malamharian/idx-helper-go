# README

## Features

### Download

Fetch and download financial report files directly from the IDX API.

1. Select a **year** and **reporting period** (TW1 / TW2 / TW3 / Tahunan).
2. Optionally filter by **file type** (.xlsx, .pdf, .zip) or a **custom regex** pattern.
3. Optionally enter specific **emiten codes** (one per line) to limit the download. Leave blank to download all listed companies.
4. Click **Fetch Reports** to retrieve the list of available files.
5. Choose a **download directory**, then click **Start All** or start individual companies.

Downloads run concurrently and can be cancelled at any point. Use the **Concurrency** dropdown to control how many files download simultaneously (be careful to avoid rate limit. Defaults to 5).

### Aggregate

Merge all `.xlsx` files from a directory into a single spreadsheet, grouped by sheet name.

1. Pick an **input directory** containing the downloaded `.xlsx` files.
2. Pick an **output file** path for the merged result.
3. Click **Aggregate** to start. Each row in the output is prefixed with the source filename.

Sheets named `Context` and `InlineXBRL` are automatically excluded.

## Important Note
If you're on Windows, you might have to turn off Windows Defender temporarily while this app is running because Windows is stupid.
Technical details: one of the lib for request simulation gets flagged frequently by Windows Defender

## About

This is the official Wails Svelte-TS template.

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.

## CI Builds

GitHub Actions builds the desktop app on `windows-latest` and `macos-latest` with `.github/workflows/build.yml`.
It runs on pushes to `main`, pull requests, and manual dispatches, then uploads the `build/bin` outputs as workflow artifacts.

## Releases

GitHub Actions publishes a GitHub Release with `.github/workflows/release.yml` whenever you push a tag matching `v*`.
The workflow builds Windows and macOS artifacts, creates the release, and attaches the packaged files.

Example:

```bash
git tag v0.1.0
git push origin v0.1.0
```
