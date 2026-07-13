You are a Principal Software Architect and Senior Go Engineer.

Your task is to build a production-ready documentation platform named **Dockube**.

Dockube is NOT another static site generator like MkDocs or Docusaurus.

Dockube is a cloud-native documentation management platform inspired by Antora, but redesigned for Kubernetes, GitOps, enterprise documentation, and modern web technologies.

The platform should be modular, scalable, and designed for long-term extensibility.

=================================
PROJECT VISION
=================================

Dockube is a documentation platform that

• manages documentation
• renders Markdown
• supports multiple products
• supports multiple versions
• supports Git repositories
• supports local filesystem
• supports GitOps workflows
• supports Kubernetes deployment
• supports enterprise multi-tenancy

Unlike Antora, Dockube should NOT generate static HTML.

Instead it should:

- render documentation dynamically
- cache rendered pages
- allow instant updates
- support server-side rendering
- use HTMX for frontend interactions
- use Go templates
- avoid React/Vue/Angular

Everything should remain lightweight.

=================================
TECH STACK
=================================

Backend

- Go 1.24+
- Chi Router
- HTMX
- html/template
- SQLite (development)
- PostgreSQL (production)
- sqlc
- Goose migrations
- Viper configuration
- Zap logging

Frontend

- HTMX
- Alpine.js (only where necessary)
- TailwindCSS
- Go templates

Markdown

goldmark

Extensions

- Tables
- Admonitions
- Task lists
- TOC
- Syntax Highlighting
- Mermaid
- Footnotes

Git

go-git

Search

Bleve

Caching

ristretto

Assets

embedded using Go embed

=================================
ARCHITECTURE
=================================

Use clean architecture.

internal/

    app/

    api/

    auth/

    cache/

    config/

    database/

    docs/

    git/

    markdown/

    renderer/

    search/

    storage/

    templates/

    users/

    workspace/

    version/

    permissions/

pkg/

cmd/

web/

migrations/

=================================
CORE CONCEPTS
=================================

Dockube manages

Workspace
    contains many products

Product
    example:

    Kubernetes
    API Gateway
    Platform
    SDK

Each Product contains

multiple versions

example

Product

Kubernetes

Versions

1.0
1.1
1.2
2.0
latest

Each version points to a source

Git repository

or

filesystem

=================================
DOCUMENT STRUCTURE
=================================

Each version contains Markdown

Example

docs/

index.md

getting-started.md

installation.md

advanced/

configuration.md

security.md

networking.md

=================================
MARKDOWN FEATURES
=================================

Support

# headings

tables

code blocks

copy buttons

syntax highlighting

Mermaid diagrams

task lists

emoji

admonitions

footnotes

automatic TOC

anchor links

cross page links

=================================
MULTI VERSION SUPPORT
=================================

Each page can switch versions.

Example

Installation

Version selector

latest

2.0

1.5

1.0

Changing version opens same page if available.

Otherwise redirect to closest page.

=================================
NAVIGATION
=================================

Sidebar generated automatically.

Support

nested folders

ordering

hidden pages

category pages

Next

Previous

Breadcrumbs

=================================
SEARCH
=================================

Full text search

Filters

workspace

product

version

page

highlight matches

=================================
URL STRUCTURE
=================================

/workspace/product/version/page

Examples

/platform/kubernetes/latest

/platform/kubernetes/latest/install

/platform/api/2.0/authentication

=================================
WORKSPACES
=================================

Workspace

contains

Products

Users

Permissions

Themes

Settings

=================================
MULTI TENANCY
=================================

Support

Organizations

Teams

Projects

Roles

Admin

Editor

Viewer

Anonymous

=================================
AUTHENTICATION
=================================

Support

Local login

LDAP

OIDC

OAuth

GitHub

GitLab

=================================
PERMISSIONS
=================================

Workspace level

Product level

Version level

Page level

=================================
RENDERING
=================================

Markdown should render on demand.

Implement cache.

Only re-render modified pages.

=================================
WATCHERS
=================================

Filesystem watcher

Git webhook

Periodic sync

=================================
GIT SUPPORT
=================================

Clone repository

Pull updates

Branches

Tags

Authentication

SSH

HTTPS

Deploy keys

=================================
IMPORTERS
=================================

Filesystem

Git

ZIP

=================================
API
=================================

REST API

Workspace CRUD

Product CRUD

Version CRUD

Search

Git sync

Webhook

=================================
HTMX UI
=================================

Entire UI should use HTMX.

No SPA.

Pages should update via partial rendering.

Use fragments.

=================================
ADMIN PANEL
=================================

Dashboard

Workspace Management

Products

Versions

Repositories

Users

Permissions

Logs

Settings

=================================
EDITOR
=================================

Built-in Markdown editor

Split preview

Autosave

Upload images

=================================
MEDIA
=================================

Image uploads

Attachments

Asset manager

=================================
THEMES
=================================

Support themes

Default

Dark

Light

Custom CSS

=================================
PLUGINS
=================================

Create plugin interface

Search providers

Markdown extensions

Authentication

Storage

Importers

=================================
OBSERVABILITY
=================================

Prometheus metrics

Health endpoint

OpenTelemetry

Structured logs

=================================
DEPLOYMENT
=================================

Docker

Docker Compose

Helm Chart

Kubernetes manifests

=================================
CI/CD
=================================

GitHub Actions

GitLab CI

Tests

Lint

Security scan

Container build

=================================
TESTING
=================================

Unit tests

Integration tests

Golden markdown rendering tests

=================================
PERFORMANCE
=================================

Use

goroutines

worker pools

cache

incremental indexing

=================================
FUTURE FEATURES
=================================

AI search

AI summarization

Documentation analytics

Page insights

Version diff

Git blame

Comments

Review workflow

Approval workflow

Knowledge graph

=================================
OUTPUT REQUIREMENTS
=================================

Build Dockube incrementally.

Do NOT dump all code at once.

Follow these phases.

Phase 1

Project structure

Phase 2

Configuration

Phase 3

Database

Phase 4

Models

Phase 5

Markdown renderer

Phase 6

Filesystem loader

Phase 7

Git loader

Phase 8

Sidebar generation

Phase 9

Template engine

Phase 10

HTMX pages

Phase 11

Search

Phase 12

Authentication

Phase 13

Admin UI

Phase 14

Caching

Phase 15

Kubernetes deployment

Each phase must include

- architecture decisions
- folder structure
- implementation
- tests
- improvements

Always produce production-quality code.

Never use shortcuts.

Follow idiomatic Go.

Prefer interfaces over concrete implementations.

Document every package.

Build this as an enterprise-grade open source project.
