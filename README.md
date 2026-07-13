# Dockube

Dockube is a self-hosted, Antora-inspired documentation portal for versioned Markdown.

## Run

```sh
go run ./cmd/server
```

Open `http://localhost:8080`. Set `DOCKUBE_IMPORT_ON_START=false` to skip the
initial import. The database defaults to `./data/dockube.db`; override it with
`DOCKUBE_DB_PATH`.

## Content catalog

Dockube loads `dockube.yml` by default (override its location with
`DOCKUBE_CONFIG`). It supports multiple local documentation components and
versions:

```yaml
site:
  title: Engineering Docs
content:
  sources:
    - url: ./services/api
      component: api
      title: Public API
      version: 2.0
      start_path: modules/ROOT/pages
      nav:
        - index.md
        - guide/getting-started.md
    - url: ./services/cli
      component: cli
      version: 1.4
```

Each source can instead contain an Antora-compatible metadata subset in
`antora.yml`; `name`, `title`, `version`, and `nav` fill in missing catalog
values. When present, `modules/ROOT/pages` is automatically used as its Markdown
root. `url` is currently a local directory; Git sources remain the next extension.

For backwards compatibility, if `dockube.yml` does not exist Dockube imports the
single directory configured by `DOCKUBE_SOURCE_DIR`, `DOCKUBE_PRODUCT`, and
`DOCKUBE_VERSION`.

## Development

```sh
go test ./...
go build ./cmd/server
```

`POST /admin/import` starts an import and requires the CSRF token issued in the
`dockube_csrf` cookie. Git webhooks and external authentication are deliberately
extension points in this first release.
