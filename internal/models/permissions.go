// Package models defines permission and user entities for multi-tenant access control.
package models

import (
	"context"
	"time"
)

// Role defines the permission level within the system.
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleEditor    Role = "editor"
	RoleViewer    Role = "viewer"
	RoleAnonymous Role = "anonymous"
)

// Permission represents a specific action that can be performed.
type Permission string

const (
	PermWorkspaceCreate Permission = "workspace:create"
	PermWorkspaceRead   Permission = "workspace:read"
	PermWorkspaceUpdate Permission = "workspace:update"
	PermWorkspaceDelete Permission = "workspace:delete"

	PermProductCreate Permission = "product:create"
	PermProductRead   Permission = "product:read"
	PermProductUpdate Permission = "product:update"
	PermProductDelete Permission = "product:delete"

	PermVersionCreate Permission = "version:create"
	PermVersionRead   Permission = "version:read"
	PermVersionUpdate Permission = "version:update"
	PermVersionDelete Permission = "version:delete"

	PermDocumentCreate Permission = "document:create"
	PermDocumentRead   Permission = "document:read"
	PermDocumentUpdate Permission = "document:update"
	PermDocumentDelete Permission = "document:delete"

	PermUserInvite  Permission = "user:invite"
	PermUserManage  Permission = "user:manage"
	PermSettingsManage Permission = "settings:manage"
)

// User represents a platform user.
type User struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email" validate:"required,email"`
	Name        string     `json:"name"`
	Avatar      string     `json:"avatar,omitempty"`
	Provider    string     `json:"provider"` // local, ldap, oidc, oauth, github, gitlab
	ProviderID  string     `json:"provider_id"`
	Preferences Preferences `json:"preferences"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// Preferences holds user-specific settings.
type Preferences struct {
	Theme           string `json:"theme"`
	Language        string `json:"language"`
	Timezone        string `json:"timezone"`
	EmailNotifications bool `json:"email_notifications"`
}

// Membership represents a user's role within a workspace.
type Membership struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	WorkspaceID int64     `json:"workspace_id"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   int64     `json:"created_by"`
}

// PermissionGrant represents a specific permission assignment.
type PermissionGrant struct {
	ID          int64      `json:"id"`
	PrincipalID int64      `json:"principal_id"` // User or Team ID
	PrincipalType string   `json:"principal_type"` // user, team
	ResourceType  string   `json:"resource_type"`  // workspace, product, version, document
	ResourceID    int64    `json:"resource_id"`
	Permission    Permission `json:"permission"`
	GrantedAt   time.Time  `json:"granted_at"`
	GrantedBy   int64      `json:"granted_by"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Team represents a group of users for permission management.
type Team struct {
	ID          int64     `json:"id"`
	WorkspaceID int64     `json:"workspace_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TeamMembership links users to teams.
type TeamMembership struct {
	TeamID    int64     `json:"team_id"`
	UserID    int64     `json:"user_id"`
	Role      Role      `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

// UserStore defines user-related operations.
type UserStore interface {
	Create(ctx context.Context, u *User) error
	Get(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id int64) error
	UpdateLastLogin(ctx context.Context, id int64) error
}

// MembershipStore defines membership operations.
type MembershipStore interface {
	Grant(ctx context.Context, m *Membership) error
	Revoke(ctx context.Context, userID, workspaceID int64) error
	Get(ctx context.Context, userID, workspaceID int64) (*Membership, error)
	ListByUser(ctx context.Context, userID int64) ([]*Membership, error)
	ListByWorkspace(ctx context.Context, workspaceID int64, role Role, limit, offset int) ([]*Membership, error)
	UpdateRole(ctx context.Context, userID, workspaceID int64, role Role) error
}

// PermissionStore defines permission operations.
type PermissionStore interface {
	Grant(ctx context.Context, g *PermissionGrant) error
	Revoke(ctx context.Context, id int64) error
	Check(ctx context.Context, userID int64, resourceType string, resourceID int64, perm Permission) (bool, error)
	ListForUser(ctx context.Context, userID int64) ([]*PermissionGrant, error)
	ListForResource(ctx context.Context, resourceType string, resourceID int64) ([]*PermissionGrant, error)
}

// TeamStore defines team operations.
type TeamStore interface {
	Create(ctx context.Context, t *Team) error
	Get(ctx context.Context, id int64) (*Team, error)
	Update(ctx context.Context, t *Team) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, workspaceID int64) ([]*Team, error)
	AddMember(ctx context.Context, membership *TeamMembership) error
	RemoveMember(ctx context.Context, teamID, userID int64) error
	ListMembers(ctx context.Context, teamID int64) ([]*TeamMembership, error)
}