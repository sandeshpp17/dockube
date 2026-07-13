// Package models defines product and version entities for the documentation platform.
package models

import (
	"context"
	"time"
)

// Product represents a documentation collection within a workspace.
// Examples: Kubernetes, API Gateway, Platform SDK.
type Product struct {
	ID          int64     `json:"id"`
	WorkspaceID int64     `json:"workspace_id"`
	Slug        string    `json:"slug" validate:"required,alphanum,min=2,max=50"`
	Name        string    `json:"name" validate:"required,min=2,max=100"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	Order       int       `json:"order"`
	Metadata    Metadata  `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Metadata holds product-specific configuration.
type Metadata struct {
	Tags         []string          `json:"tags,omitempty"`
	Owners       []string          `json:"owners,omitempty"`
	Repository   string            `json:"repository,omitempty"`
	Homepage     string            `json:"homepage,omitempty"`
	ContactEmail string            `json:"contact_email,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// ProductVersion represents a specific version of a product.
// Each version links to a source (Git or filesystem) containing the documentation.
type ProductVersion struct {
	ID           int64     `json:"id"`
	ProductID    int64     `json:"product_id"`
	Version      string    `json:"version" validate:"required"`
	DisplayName  string    `json:"display_name,omitempty"`
	SourceType   string    `json:"source_type" validate:"required,oneof=git filesystem zip"`
	SourceURL    string    `json:"source_url"`
	SourceBranch string    `json:"source_branch,omitempty"`
	SourcePath   string    `json:"source_path,omitempty"`
	IsStable     bool      `json:"is_stable"`
	IsLatest     bool      `json:"is_latest"`
	Status       string    `json:"status" validate:"oneof=active deprecated archived"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	SyncedAt     *time.Time `json:"synced_at,omitempty"`
}

// VersionAlias provides human-friendly names for versions (e.g., "latest", "stable").
type VersionAlias struct {
	ProductVersionID int64  `json:"product_version_id"`
	Alias            string `json:"alias" validate:"required"`
}

// Document represents a single documentation page.
type Document struct {
	ID             int64     `json:"id"`
	ProductVersionID int64   `json:"product_version_id"`
	Path           string    `json:"path"`
	Title          string    `json:"title"`
	Owner          string    `json:"owner,omitempty"`
	Source         string    `json:"source"` // Raw markdown source
	HTML           string    `json:"html"`   // Rendered HTML
	Tags           []string  `json:"tags,omitempty"`
	Order          int       `json:"order"`
	ParentPath     string    `json:"parent_path,omitempty"`
	IsHidden       bool      `json:"is_hidden"`
	Metadata       DocMetadata `json:"metadata"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DocMetadata holds document-specific information.
type DocMetadata struct {
	Authors      []string          `json:"authors,omitempty"`
	LastModified time.Time         `json:"last_modified,omitempty"`
	ReadingTime  int               `json:"reading_time,omitempty"` // minutes
	WordCount    int               `json:"word_count,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// SearchResult represents a search match with context.
type SearchResult struct {
	Product      string `json:"product"`
	Version      string `json:"version"`
	Path         string `json:"path"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet"`
	Score        float64 `json:"score"`
	HighlightPositions []int `json:"highlight_positions,omitempty"`
}

// ProductStore defines the interface for product operations.
type ProductStore interface {
	Create(ctx context.Context, p *Product) error
	Get(ctx context.Context, id int64) (*Product, error)
	GetBySlug(ctx context.Context, workspaceID int64, slug string) (*Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, workspaceID int64, limit, offset int) ([]*Product, error)
}

// VersionStore defines the interface for version operations.
type VersionStore interface {
	Create(ctx context.Context, v *ProductVersion) error
	Get(ctx context.Context, id int64) (*ProductVersion, error)
	GetByVersion(ctx context.Context, productID int64, version string) (*ProductVersion, error)
	Update(ctx context.Context, v *ProductVersion) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, productID int64) ([]*ProductVersion, error)
	ResolveAlias(ctx context.Context, productID int64, alias string) (*ProductVersion, error)
	SetAlias(ctx context.Context, versionID int64, alias string) error
}

// DocumentStore defines the interface for document operations.
type DocumentStore interface {
	Upsert(ctx context.Context, d *Document) error
	Get(ctx context.Context, versionID int64, path string) (*Document, error)
	List(ctx context.Context, versionID int64, parentPath string) ([]*Document, error)
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, versionID int64, query string, filters SearchFilters) ([]*SearchResult, error)
	SearchAll(ctx context.Context, query string, workspaceID int64) ([]*SearchResult, error)
}

// SearchFilters provides search refinement options.
type SearchFilters struct {
	Tags   []string
	Owner  string
	Limit  int
	Offset int
}