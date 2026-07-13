# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Dockube is a self-hosted, Antora-inspired documentation portal for versioned Markdown, written in Go. A single server process imports Markdown files from local directories into SQLite, then serves them through server-rendered HTML with htmx-powered partial updates.

## Commands

```sh
go run ./cmd/server        # run the server (http://localhost:8080)
go build ./cmd/server      # build
go test ./...              # run all tests
go test ./internal/handlers -run TestDocumentSearchAndCSRF   # run a single test
go vet ./...                # static checks
```

Environment variables (all optional, see `internal/config/config.go`):
- `DOCKUBE_ADDR` (default `:8080`)
- `DOCKUBE_DB_PATH` (default `data/dockube.db`)
- `DOCKUBE_CONFIG` (default `dockube.yml`) — path to the content catalog
- `DOCKUBE_IMPORT_ON_START` (default `true`) — set `false` to skip the startup import
- `DOCKUBE_SOURCE_DIR` / `DOCKUBE_PRODUCT` / `DOCKUBE_VERSION` — legacy single-source fallback used only when `dockube.yml` is absent

There is no separate lint config; `go vet` is the available static check.

## Architecture

Request flow: `cmd/server/main.go` wires everything together — it loads config, opens the DB (which runs embedded migrations), builds the list of `importer.Job`s from the catalog, optionally runs an initial import in a goroutine, and starts the chi router from `handlers.App`.

### Content pipeline (catalog → importer → markdown → models/db)

1. **`internal/catalog`** parses `dockube.yml` (Antora-inspired). Each `content.sources` entry describes a local directory (`url`), becomes one "component" (`component`/`slug`) at one `version`. If the source directory contains an `antora.yml`, `Source.applyAntora()` fills in missing `component`/`title`/`version`/`nav` from it and auto-detects `modules/ROOT/pages` as the Markdown root. `url` is always a local directory today — Git sources are a known future extension (see `importer.GitSource`).
2. **`internal/importer`** (`Importer.Run`) walks a source directory for `*.md` files, strips the extension to build each document's `path` (an `index.md` file becomes path `""`), renders each file via `internal/markdown`, and upserts the result into the DB via `models.Store`.
3. **`internal/markdown`** (`Render`) parses a small YAML-like front-matter block (`title`, `owner`, `tags`), rewrites `[[wiki-style]]` links into `/docs/{product}/{version}/{path}` URLs, then renders the body with goldmark (GFM + syntax highlighting). Mermaid code fences are left as inert fenced blocks for client-side rendering — no scripts are ever generated from Markdown content.
4. **`internal/models`** (`Store`) is the only SQL layer. It manages `products` → `product_versions` (with `version_aliases` like `latest`/`stable`) → `documents` (+ `document_tags` and an FTS5 `documents_fts` table for search). All queries live directly on `Store` methods — there is no ORM.
5. **`internal/db`** owns schema migrations. Migrations are embedded from `internal/db/migrations/*.sql` (numbered `NNN_name.sql`) and applied in order, tracked in a `schema_migrations` table, each in its own transaction. **The top-level `migrations/` directory is a duplicate kept in sync with `internal/db/migrations/` for reference — when adding a migration, add the file to `internal/db/migrations/` (the embedded copy that actually runs) and mirror it in `migrations/`.**

### Web layer (`internal/handlers`)

- Single `App` struct holds `Store`, `Importer`, `ImportJobs`, and a `Navigation` map (`"product@version"` → configured nav paths from the catalog).
- Routing uses `go-chi`. Key routes: `/` (component list), `/docs/{product}/{version}/*` (a doc page, resolving `version` through aliases via `Store.ResolveVersion`), `/search` (global FTS search across all components/versions), `/search/{product}/{version}` (scoped search), `POST /admin/import` (re-run all import jobs).
- HTML templates are plain `html/template` (not `text/template`), defined as Go string literals directly in `handlers.go` — there is no separate `.html` template file for the live app. `web/templates/app.templ` exists only as a placeholder for a future `templ`-based rewrite; it is not currently compiled or used.
- htmx drives partial rendering: handlers check `HX-Request` header (`partial()`) and execute only the `"content"` template block instead of the full `base` layout when true.
- CSRF protection is custom and cookie-based (`dockube_csrf`), checked via `validCSRF` (token match + same-origin) — required for `POST /admin/import`.
- `orderedNavigation` merges the catalog's configured `nav` ordering with any imported documents not explicitly listed, so pages are never silently hidden from navigation even if omitted from `nav`.

### Plugins (`internal/plugins`)

Defines extension-point interfaces (`DocProcessor`, `Renderer`, `SearchIndexer`, `AuthProvider`) and a `Registry`, plus one concrete `MermaidRenderer`. This package is largely aspirational scaffolding for future extensibility (git sync, external auth, custom rendering/indexing) — only `MermaidRenderer` is registered/used-in-spirit today; wire new plugins through `Registry` rather than ad hoc hooks in `handlers`.

## Conventions specific to this codebase

- Business logic is intentionally dense/compact (many one-line functions, minimal blank lines) — match existing style rather than expanding it when making small edits.
- `models.Store` methods take `context.Context` first and return `(T, error)` — follow this pattern for new queries.
- Tests use `t.TempDir()` plus a real SQLite DB (via `db.Open`) rather than mocks — see `internal/handlers/handlers_test.go` and `internal/importer/importer_test.go` for the pattern of building a temp source tree, importing it, then asserting on HTTP responses or store state.
