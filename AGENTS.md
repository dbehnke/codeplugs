# Codeplugs Repository Guide

## Purpose and layout

This repository contains `codeplugs`, a Go CLI and web application for importing, managing, and exporting amateur-radio codeplugs. It ingests sources such as RepeaterBook data, RadioID.net contacts, and radio CSV exports into a central SQLite database, then exports radio-specific formats. SQLite is accessed through GORM with the pure-Go `modernc.org/sqlite` driver.

- `main.go` wires CLI flags, database setup, imports/exports, and the embedded web UI.
- `models/` contains the persisted domain model.
- `importer/` and `exporter/` contain CSV and radio-specific format logic.
- `services/` contains business logic; `api/` contains HTTP and WebSocket handlers.
- `frontend/` is a Vue 3/Vite/Pinia/Tailwind CSS v4 application managed with Bun.
- `filters/`, `generated/`, `outputs/`, and the radio-named directories contain source or generated radio data. Avoid bulk-changing these files unless the task specifically concerns them.

Supported radio profiles include Radioddity DB25-D, Baofeng DM32UV, and AnyTone 890. Yaesu System Fusion support is intentionally out of scope.

## Current capabilities

- Import generic CSV, RepeaterBook data, ZIP archives, and radio-specific CSV files.
- Import and export Baofeng DM32UV channels, contacts, talkgroups, zones, and roaming data.
- Import and export AnyTone 890 data, including roaming and scan lists.
- Import and export Radioddity DB25-D and CHIRP-compatible CSV data.
- Manage channels, contacts, zones, scan lists, roaming configuration, and contact filter lists through the CLI, API, or work-in-progress web UI.
- Generate filtered RadioID.net contact lists in `radioid`, `dm32uv`, and `at890` formats.

## Development workflow

Use `task` as the canonical command runner; the `Makefile` is legacy.

```bash
task test             # all Go tests
task test-race        # Go tests with race detection
task vet              # go vet
task lint             # golangci-lint, with go vet fallback
task fmt-check        # verify Go formatting
task frontend-install # install frontend dependencies with Bun
task frontend-build   # build the Vue application
task fast-build       # build Go binary using the existing frontend/dist
task build            # build frontend, then the Go binary
task ci               # formatting, vet, lint, and Go tests
```

Run frontend tests explicitly with `cd frontend && bun test`; they are not part of `task test`. The root binary embeds `frontend/dist`, so a clean full build must generate that directory before compiling Go.

Prerequisites are Go 1.26.1 or newer, Bun, and Go Task. The Taskfile sets `GO111MODULE=on` and `CGO_ENABLED=0`.

When automating commands under zsh, unset correction options if they are enabled so prompts do not interfere:

```zsh
unsetopt correct_all
unsetopt correct
```

Install the repository's pre-commit hook with `./scripts/install-hooks.sh`. It runs `golangci-lint`; bypass it only in an explicitly approved emergency.

## Common operations

```bash
# Import/export a generic or DB25-D CSV
./codeplugs --import channels.csv
./codeplugs --export output.csv

# Import/export a DM32UV directory
./codeplugs --import path/to/dm32uv/ --radio dm32uv
./codeplugs --export path/to/output/ --radio dm32uv

# Import/export AnyTone 890 data
./codeplugs --import input.csv --radio at890
./codeplugs --export path/to/output/ --radio at890

# Start the web UI at http://localhost:8080
./codeplugs --serve --port 8080

# Generate a filtered contact list
./codeplugs --generate-contacts \
  --filter-file filters/my-contacts.csv \
  --source-file user.csv \
  --output-file contacts.csv \
  --contact-format dm32uv
```

`--contact-format` accepts `radioid` (default), `dm32uv`, or `at890`. Filter file details live in `filters/README.md`. `task generate-brandmeister` generates all three formats from the BrandMeister filter; it may download current RadioID.net data and create timestamped artifacts under `outputs/`.

## Change guidelines

- Follow test-driven development: add or update a focused failing test first when practical, verify the failure, implement the smallest change, then refactor. Maintain strong coverage of importers, exporters, API handlers, and services.
- Keep radio-specific parsing and serialization in the corresponding importer/exporter package. Preserve exact CSV headers, quoting, line endings, ordering, and CPS limits; these formats are compatibility contracts.
- Keep database concerns in `database/` and `models/`, business rules in `services/`, and transport concerns in `api/`.
- Add tests beside the affected Go package using table-driven cases where useful. Frontend tests use Vitest and Vue Test Utils.
- Format Go changes with `gofmt`/`task fmt`. Follow the existing Vue composition style and avoid introducing a second package manager.
- Do not commit local databases, the `codeplugs` binary, `frontend/dist`, downloaded contact databases, or generated `outputs/` artifacts unless the task explicitly requires an artifact update.
- Do not bypass or weaken tests, lint rules, or the pre-commit hook to make a change pass.

## Releases and generated artifacts

- Version tags such as `v1.2.3` trigger GoReleaser builds for supported Linux, macOS, and Windows architectures. Use `task release` for a local snapshot or `task release-local` for the current platform.
- Filter changes can trigger automated contact generation. Review generated contact counts and all three output formats before accepting them.
- Treat radio sample data and generated CSV output as potentially large compatibility fixtures. Make targeted edits and call out intentional fixture regeneration.

## BrandMeister automation

The scheduled workflow in `.github/workflows/brandmeister-fetch.yml` uses `scripts/fetch-brandmeister.ts` and Playwright to query BrandMeister before generating release artifacts.

- Use the current hash route `https://brandmeister.network/#/contactsexport`. The legacy `?page=contactsexport` URL redirects to the User Dashboard and does not render the export controls.
- Prefer the page's stable element IDs: `#talkgroups` for input, `#addTalkgroup` for Run, and `#userTable` for results. Avoid generic selectors such as `input[type="text"]` because the site contains unrelated controls and its layout can change.
- The results table and CSV button exist before a query completes. Do not use CSV-button visibility/enabled state or a fixed sleep as a completion signal. Wait for a real result cell to replace DataTables' `.dt-empty` placeholder before exporting.
- Create the Playwright download waiter immediately before clicking CSV so query time does not consume the download timeout.
- Always close the browser in a `finally` block so retry attempts do not leak Chromium processes.
- Diagnose remote failures with `gh run list --workflow brandmeister-fetch.yml` and `gh run view <run-id> --log-failed`. Compare multiple recent failures before assuming a transient outage.
- When changing the scraper, validate it end-to-end against the live page using a temporary output path. Confirm the CSV header and a plausible nonzero record count without modifying tracked filter or output fixtures.
- Treat BrandMeister DOM structure and routes as external compatibility contracts that can change without a repository change. Inspect the current rendered page before adjusting timeouts or retry counts.

## Direction

Expected future work includes improving drag-and-drop zone/channel management, adding appropriate radio formats, direct repeater-database integration, and contact synchronization. Preserve the explicit exclusion of Yaesu System Fusion unless project scope is deliberately changed.

## Validation

Run the narrowest relevant tests while iterating, then validate the affected stack before handoff:

- Go-only changes: `go test ./path/to/package` followed by `task ci` when feasible.
- Frontend-only changes: `cd frontend && bun test` and `task frontend-build`.
- Cross-stack or embedding changes: `task build` plus the relevant Go and frontend tests.

Report any checks that could not be run and why.
