# README

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
