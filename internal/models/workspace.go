// Package models defines the core domain entities for Dockube's multi-tenant documentation platform.
// This file implements the Workspace model which is the top-level tenant container.
package models

import (
	"context"
	"time"
)

// Workspace represents a top-level organizational unit containing products, users, and settings.
// Workspaces provide isolation for multi-tenant deployments.
type Workspace struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug" validate:"required,alphanum,min=2,max=50"`
	Name        string    `json:"name" validate:"required,min=2,max=100"`
	Description string    `json:"description,omitempty"`
	Settings    Settings  `json:"settings"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   int64     `json:"created_by"`
}

// Settings holds workspace-level configuration.
type Settings struct {
	Theme           string            `json:"theme"`
	CustomCSS       string            `json:"custom_css,omitempty"`
	Features        map[string]bool   `json:"features"`
	DefaultVersion  string            `json:"default_version"`
	SearchEnabled   bool              `json:"search_enabled"`
	CommentsEnabled bool              `json:"comments_enabled"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// NewWorkspace creates a new workspace with default settings.
func NewWorkspace(slug, name string) *Workspace {
	return &Workspace{
		Slug: slug,
		Name: name,
		Settings: Settings{
			Theme:           "default",
			SearchEnabled:   true,
			CommentsEnabled: false,
			Features: map[string]bool{
				"git_sync":         true,
				"webhook_support":  true,
				"advanced_search":  true,
				"version_switch":   true,
				"toc_generation":   true,
				"mermaid_diagrams": true,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// WorkspaceStore defines the interface for workspace data operations.
type WorkspaceStore interface {
	Create(ctx context.Context, ws *Workspace) error
	Get(ctx context.Context, id int64) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	Update(ctx context.Context, ws *Workspace) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*Workspace, error)
	Count(ctx context.Context) (int64, error)
}