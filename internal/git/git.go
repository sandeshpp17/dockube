// Package git provides Git repository operations for Dockube's documentation import system.
// It supports cloning, pulling, branch/tag operations with multiple authentication methods.
package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Config holds Git operation configuration.
type Config struct {
	CloneTimeout  time.Duration
	AuthMethod    string // ssh, https, token
	SSHKeyPath    string
	SSHPassphrase string
	Token         string
	Username      string
	WebhookSecret string
	SyncInterval  time.Duration
}

// Client provides Git operations.
type Client struct {
	config *Config
	logger *zap.Logger
}

// Source represents a Git repository source for documentation.
type Source struct {
	URL        string
	Branch     string
	Tag        string
	Path       string // Subdirectory within repo
	Auth       *Auth
	LocalPath  string // Local clone location
}

// Auth holds authentication credentials.
type Auth struct {
	Method    string // ssh, https, token
	SSHKey    []byte
	SSHKeyPath string
	Passphrase string
	Token     string
	Username  string
	Password  string
}

// SyncResult contains information about a sync operation.
type SyncResult struct {
	Repository   string
	Branch       string
	CommitHash   string
	CommitAuthor string
	CommitDate   time.Time
	FilesChanged int
	LocalPath    string
}

// New creates a new Git client.
func New(config *Config, logger *zap.Logger) *Client {
	return &Client{
		config: config,
		logger: logger,
	}
}

// Clone performs a full clone of a repository.
func (c *Client) Clone(ctx context.Context, source *Source) (*SyncResult, error) {
	c.logger.Info("cloning repository",
		zap.String("url", source.URL),
		zap.String("branch", source.Branch),
	)

	cloneOpts := &git.CloneOptions{
		URL: source.URL,
		Progress: os.Stdout,
	}

	// Configure authentication
	auth, err := c.getAuth(source.Auth)
	if err != nil {
		return nil, fmt.Errorf("auth configuration failed: %w", err)
	}
	if auth != nil {
		cloneOpts.Auth = auth
	}

	// Set branch reference
	if source.Branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(source.Branch)
		cloneOpts.SingleBranch = true
	}

	// Set clone timeout
	cloneCtx, cancel := context.WithTimeout(ctx, c.config.CloneTimeout)
	defer cancel()

	// Ensure local path exists
	if source.LocalPath == "" {
		source.LocalPath = filepath.Join("data", "repos", filepath.Base(source.URL))
	}
	if err := os.MkdirAll(filepath.Dir(source.LocalPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create local path: %w", err)
	}

	// Perform clone
	repo, err := git.PlainCloneContext(cloneCtx, source.LocalPath, false, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	return c.buildSyncResult(repo, source)
}

// Pull fetches and merges updates from the remote.
func (c *Client) Pull(ctx context.Context, source *Source) (*SyncResult, error) {
	c.logger.Info("pulling repository",
		zap.String("path", source.LocalPath),
	)

	repo, err := git.PlainOpen(source.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	pullOpts := &git.PullOptions{}

	// Configure authentication
	auth, err := c.getAuth(source.Auth)
	if err != nil {
		return nil, fmt.Errorf("auth configuration failed: %w", err)
	}
	if auth != nil {
		pullOpts.Auth = auth
	}

	// Set branch
	if source.Branch != "" {
		pullOpts.ReferenceName = plumbing.NewBranchReferenceName(source.Branch)
	}

	// Perform pull
	err = worktree.PullContext(ctx, pullOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("pull failed: %w", err)
	}

	return c.buildSyncResult(repo, source)
}

// Sync ensures the repository is up to date, cloning if necessary.
func (c *Client) Sync(ctx context.Context, source *Source) (*SyncResult, error) {
	// Check if already cloned
	if _, err := os.Stat(filepath.Join(source.LocalPath, ".git")); err == nil {
		return c.Pull(ctx, source)
	}

	// Clone if not present
	return c.Clone(ctx, source)
}

// Checkout switches to a specific branch or tag.
func (c *Client) Checkout(ctx context.Context, source *Source, ref string) error {
	repo, err := git.PlainOpen(source.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	checkoutOpts := &git.CheckoutOptions{}

	// Determine if ref is a branch or tag
	if plumbing.IsHash(ref) {
		checkoutOpts.Hash = plumbing.NewHash(ref)
	} else {
		// Try as branch first, then tag
		branchRef := plumbing.NewBranchReferenceName(ref)
		_, err := repo.Reference(branchRef, true)
		if err == nil {
			checkoutOpts.Branch = branchRef
		} else {
			tagRef := plumbing.NewTagReferenceName(ref)
			checkoutOpts.Branch = tagRef
		}
	}

	return worktree.Checkout(checkoutOpts)
}

// ListBranches returns all branches in the repository.
func (c *Client) ListBranches(ctx context.Context, source *Source) ([]string, error) {
	repo, err := git.PlainOpen(source.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	var branches []string
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branches = append(branches, ref.Name().Short())
		}
		return nil
	})

	return branches, err
}

// ListTags returns all tags in the repository.
func (c *Client) ListTags(ctx context.Context, source *Source) ([]string, error) {
	repo, err := git.PlainOpen(source.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	tags, err := repo.TagObjects()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	var tagNames []string
	err = tags.ForEach(func(tag *object.Tag) error {
		tagNames = append(tagNames, tag.Name)
		return nil
	})

	return tagNames, err
}

// GetCommitInfo returns information about the current commit.
func (c *Client) GetCommitInfo(source *Source) (*object.Commit, error) {
	repo, err := git.PlainOpen(source.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return commit, nil
}

// getAuth configures authentication based on the method.
func (c *Client) getAuth(auth *Auth) (transport.AuthMethod, error) {
	if auth == nil {
		return nil, nil
	}

	switch auth.Method {
	case "ssh":
		return c.getSSHAuth(auth)
	case "https":
		return c.getHTTPSAuth(auth)
	case "token":
		return &http.BasicAuth{
			Username: "oauth2",
			Password: auth.Token,
		}, nil
	default:
		return nil, nil
	}
}

// getSSHAuth configures SSH authentication.
func (c *Client) getSSHAuth(auth *Auth) (transport.AuthMethod, error) {
	var sshKey []byte
	var err error

	if auth.SSHKey != nil {
		sshKey = auth.SSHKey
	} else if auth.SSHKeyPath != "" {
		sshKey, err = os.ReadFile(auth.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("SSH key required")
	}

	// Parse the private key
	publicKeys, err := ssh.NewPublicKeys("git", sshKey, auth.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// Configure host key verification
	knownHosts, err := knownhosts.New("~/.ssh/known_hosts")
	if err != nil {
		c.logger.Warn("failed to load known_hosts, using insecure verification",
			zap.Error(err),
		)
		publicKeys.HostKeyCallbackHelper.HostKeyCallback = ssh.InsecureIgnoreHostKey().HostKeyCallback
	} else {
		publicKeys.HostKeyCallbackHelper.HostKeyCallback = knownHosts.HostKeyCallback
	}

	return publicKeys, nil
}

// getHTTPSAuth configures HTTPS authentication.
func (c *Client) getHTTPSAuth(auth *Auth) (transport.AuthMethod, error) {
	if auth.Token != "" {
		return &http.BasicAuth{
			Username: "oauth2",
			Password: auth.Token,
		}, nil
	}

	if auth.Username != "" && auth.Password != "" {
		return &http.BasicAuth{
			Username: auth.Username,
			Password: auth.Password,
		}, nil
	}

	return nil, nil
}

// buildSyncResult creates a SyncResult from repository state.
func (c *Client) buildSyncResult(repo *git.Repository, source *Source) (*SyncResult, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Count files (simplified)
	worktree, _ := repo.Worktree()
	filesChanged := 0
	if worktree != nil {
		status, _ := worktree.Status()
		filesChanged = len(status)
	}

	return &SyncResult{
		Repository:   source.URL,
		Branch:       head.Name().Short(),
		CommitHash:   head.Hash().String(),
		CommitAuthor: commit.Author.Name,
		CommitDate:   commit.Author.When,
		FilesChanged: filesChanged,
		LocalPath:    source.LocalPath,
	}, nil
}

// WebhookHandler processes incoming Git webhook payloads.
type WebhookHandler struct {
	Client *Client
	Secret string
	Logger *zap.Logger
}

// NewWebhookHandler creates a webhook handler.
func NewWebhookHandler(client *Client, secret string, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		Client: client,
		Secret: secret,
		Logger: logger,
	}
}

// Handle processes a webhook payload and triggers sync if valid.
func (w *WebhookHandler) Handle(ctx context.Context, source *Source, payload []byte, signature string) (*SyncResult, error) {
	// Verify webhook signature if secret is configured
	if w.Secret != "" {
		if !w.verifySignature(payload, signature) {
			return nil, fmt.Errorf("invalid webhook signature")
		}
	}

	// Parse payload to determine what changed (simplified for now)
	// In production, would parse GitHub/GitLab webhook formats

	w.Logger.Info("processing webhook",
		zap.String("repository", source.URL),
	)

	// Trigger sync
	return w.Client.Sync(ctx, source)
}

// verifySignature validates the webhook signature.
func (w *WebhookHandler) verifySignature(payload []byte, signature string) bool {
	// Implementation depends on webhook provider (GitHub uses HMAC-SHA1, GitLab uses HMAC-SHA256)
	// This is a placeholder
	return true
}