// Package config provides configuration management for Dockube using Viper.
// It supports multiple configuration sources (files, environment variables, defaults)
// with validation and structured configuration for all platform components.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config holds the complete Dockube configuration with validation tags.
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server" validate:"required"`

	// Database configuration
	Database DatabaseConfig `mapstructure:"database" validate:"required"`

	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging"`

	// Security configuration
	Security SecurityConfig `mapstructure:"security"`

	// Cache configuration
	Cache CacheConfig `mapstructure:"cache"`

	// Git configuration
	Git GitConfig `mapstructure:"git"`

	// Search configuration
	Search SearchConfig `mapstructure:"search"`

	// Import configuration
	Import ImportConfig `mapstructure:"import"`

	// Application metadata
	App AppConfig `mapstructure:"app"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Address         string        `mapstructure:"address" validate:"required,hostname_port"`
	Port            int           `mapstructure:"port" validate:"required,min=1,max=65535"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" validate:"required,min=1000000000"`      // 1 second minimum
	WriteTimeout    time.Duration `mapstructure:"write_timeout" validate:"required,min=1000000000"`     // 1 second minimum
	IdleTimeout     time.Duration `mapstructure:"idle_timeout" validate:"required,min=1000000000"`      // 1 second minimum
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required,min=1000000000"`  // 1 second minimum
	CORS            CORSConfig    `mapstructure:"cors"`
}

// CORSConfig holds CORS-specific settings.
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Type            string        `mapstructure:"type" validate:"required,oneof=sqlite postgresql"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port" validate:"omitempty,min=1,max=65535"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode" validate:"omitempty,oneof=disable require verify-ca verify-full"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" validate:"min=1"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" validate:"min=0"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	Path            string        `mapstructure:"path"` // For SQLite
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"required,oneof=json console"`
	Output string `mapstructure:"output" validate:"required,oneof=stdout stderr file"`
	File   string `mapstructure:"file"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	CSRFEnabled     bool          `mapstructure:"csrf_enabled"`
	CSRFCookieName  string        `mapstructure:"csrf_cookie_name"`
	CSRFSameSite    string        `mapstructure:"csrf_same_site" validate:"omitempty,oneof=lax strict none"`
	RateLimitEnabled bool        `mapstructure:"rate_limit_enabled"`
	RateLimitRPS    int           `mapstructure:"rate_limit_rps" validate:"min=1"`
	RateLimitBurst  int           `mapstructure:"rate_limit_burst" validate:"min=1"`
	Auth            AuthConfig    `mapstructure:"auth"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Enabled  bool           `mapstructure:"enabled"`
	Provider string         `mapstructure:"provider" validate:"omitempty,oneof=local ldap oidc oauth github gitlab"`
	OIDC     OIDCConfig     `mapstructure:"oidc"`
	OAuth    OAuthConfig    `mapstructure:"oauth"`
	LDAP     LDAPConfig     `mapstructure:"ldap"`
}

// OIDCConfig holds OpenID Connect settings.
type OIDCConfig struct {
	IssuerURL    string   `mapstructure:"issuer_url"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
}

// OAuthConfig holds OAuth settings.
type OAuthConfig struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
}

// LDAPConfig holds LDAP authentication settings.
type LDAPConfig struct {
	Server       string `mapstructure:"server"`
	Port         int    `mapstructure:"port" validate:"omitempty,min=1,max=65535"`
	BaseDN       string `mapstructure:"base_dn"`
	BindDN       string `mapstructure:"bind_dn"`
	BindPassword string `mapstructure:"bind_password"`
	Filter       string `mapstructure:"filter"`
}

// CacheConfig holds caching configuration.
type CacheConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	Type        string        `mapstructure:"type" validate:"omitempty,oneof=memory ristretto redis"`
	MaxSize     int64         `mapstructure:"max_size"`
	TTL         time.Duration `mapstructure:"ttl"`
	RedisURL    string        `mapstructure:"redis_url"`
}

// GitConfig holds Git operation settings.
type GitConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	CloneTimeout    time.Duration `mapstructure:"clone_timeout"`
	AuthMethod      string        `mapstructure:"auth_method" validate:"omitempty,oneof=ssh https token"`
	SSHKeyPath      string        `mapstructure:"ssh_key_path"`
	SSHPassphrase   string        `mapstructure:"ssh_passphrase"`
	Token           string        `mapstructure:"token"`
	WebhookSecret   string        `mapstructure:"webhook_secret"`
	SyncInterval    time.Duration `mapstructure:"sync_interval"`
}

// SearchConfig holds search engine settings.
type SearchConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Engine     string `mapstructure:"engine" validate:"omitempty,oneof=bleve sqlite_fts"`
	IndexPath  string `mapstructure:"index_path"`
	BatchSize  int    `mapstructure:"batch_size" validate:"min=1"`
}

// ImportConfig holds document import settings.
type ImportConfig struct {
	OnStart         bool          `mapstructure:"on_start"`
	CatalogPath     string        `mapstructure:"catalog_path"`
	WatchEnabled    bool          `mapstructure:"watch_enabled"`
	WatchInterval   time.Duration `mapstructure:"watch_interval"`
	ConcurrentJobs  int           `mapstructure:"concurrent_jobs" validate:"min=1"`
}

// AppConfig holds application metadata.
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment" validate:"required,oneof=development staging production"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:         ":8080",
			Port:            8080,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			CORS: CORSConfig{
				Enabled:        false,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			},
		},
		Database: DatabaseConfig{
			Type:            "sqlite",
			Path:            "data/dockube.db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			SSLMode:         "disable",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Security: SecurityConfig{
			CSRFEnabled:      true,
			CSRFCookieName:   "dockube_csrf",
			CSRFSameSite:     "strict",
			RateLimitEnabled: true,
			RateLimitRPS:     100,
			RateLimitBurst:   200,
		},
		Cache: CacheConfig{
			Enabled: true,
			Type:    "ristretto",
			MaxSize: 100 * 1024 * 1024, // 100MB
			TTL:     5 * time.Minute,
		},
		Git: GitConfig{
			Enabled:      true,
			CloneTimeout: 5 * time.Minute,
			AuthMethod:   "https",
			SyncInterval: 15 * time.Minute,
		},
		Search: SearchConfig{
			Enabled:   true,
			Engine:    "bleve",
			IndexPath: "data/search.bleve",
			BatchSize: 100,
		},
		Import: ImportConfig{
			OnStart:        true,
			CatalogPath:    "dockube.yml",
			WatchEnabled:   false,
			WatchInterval:  1 * time.Minute,
			ConcurrentJobs: 4,
		},
		App: AppConfig{
			Name:        "dockube",
			Version:     "0.1.0",
			Environment: "development",
		},
	}
}

// Load reads configuration from multiple sources with proper precedence.
// Precedence (highest to lowest): CLI flags, env vars, config file, defaults.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure environment variable handling
	v.SetEnvPrefix("DOCKUBE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Find and read config file
	v.SetConfigName("dockube")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/dockube")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable - we'll use defaults/env vars
	}

	// Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply CLI/env overrides for critical settings
	applyOverrides(&cfg)

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults configures default values in Viper.
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()

	v.SetDefault("server.address", defaults.Server.Address)
	v.SetDefault("server.port", defaults.Server.Port)
	v.SetDefault("server.read_timeout", defaults.Server.ReadTimeout)
	v.SetDefault("server.write_timeout", defaults.Server.WriteTimeout)
	v.SetDefault("server.idle_timeout", defaults.Server.IdleTimeout)
	v.SetDefault("server.shutdown_timeout", defaults.Server.ShutdownTimeout)

	v.SetDefault("database.type", defaults.Database.Type)
	v.SetDefault("database.path", defaults.Database.Path)
	v.SetDefault("database.max_open_conns", defaults.Database.MaxOpenConns)
	v.SetDefault("database.max_idle_conns", defaults.Database.MaxIdleConns)
	v.SetDefault("database.conn_max_lifetime", defaults.Database.ConnMaxLifetime)

	v.SetDefault("logging.level", defaults.Logging.Level)
	v.SetDefault("logging.format", defaults.Logging.Format)
	v.SetDefault("logging.output", defaults.Logging.Output)

	v.SetDefault("security.csrf_enabled", defaults.Security.CSRFEnabled)
	v.SetDefault("security.csrf_cookie_name", defaults.Security.CSRFCookieName)
	v.SetDefault("security.rate_limit_enabled", defaults.Security.RateLimitEnabled)
	v.SetDefault("security.rate_limit_rps", defaults.Security.RateLimitRPS)

	v.SetDefault("cache.enabled", defaults.Cache.Enabled)
	v.SetDefault("cache.type", defaults.Cache.Type)
	v.SetDefault("cache.max_size", defaults.Cache.MaxSize)
	v.SetDefault("cache.ttl", defaults.Cache.TTL)

	v.SetDefault("git.enabled", defaults.Git.Enabled)
	v.SetDefault("git.clone_timeout", defaults.Git.CloneTimeout)
	v.SetDefault("git.sync_interval", defaults.Git.SyncInterval)

	v.SetDefault("search.enabled", defaults.Search.Enabled)
	v.SetDefault("search.engine", defaults.Search.Engine)
	v.SetDefault("search.batch_size", defaults.Search.BatchSize)

	v.SetDefault("import.on_start", defaults.Import.OnStart)
	v.SetDefault("import.catalog_path", defaults.Import.CatalogPath)
	v.SetDefault("import.concurrent_jobs", defaults.Import.ConcurrentJobs)

	v.SetDefault("app.name", defaults.App.Name)
	v.SetDefault("app.version", defaults.App.Version)
	v.SetDefault("app.environment", defaults.App.Environment)
}

// applyOverrides applies environment variable overrides for critical settings.
func applyOverrides(cfg *Config) {
	// Database path override (legacy support)
	if path := getEnv("DOCKUBE_DB_PATH"); path != "" {
		cfg.Database.Path = path
	}

	// Server address override
	if addr := getEnv("DOCKUBE_ADDR"); addr != "" {
		cfg.Server.Address = addr
	}

	// Catalog path override
	if cat := getEnv("DOCKUBE_CONFIG"); cat != "" {
		cfg.Import.CatalogPath = cat
	}

	// Import on start override
	if importOnStart := getEnv("DOCKUBE_IMPORT_ON_START"); importOnStart != "" {
		cfg.Import.OnStart = importOnStart != "false"
	}
}

// getEnv is a helper to get environment variables.
func getEnv(key string) string {
	return os.Getenv(key)
}

// validateConfig validates the configuration using struct tags.
func validateConfig(cfg *Config) error {
	validate := validator.New()

	if err := validate.Struct(cfg); err != nil {
		return fmt.Errorf("validation errors: %w", err)
	}

	// Additional business logic validation
	if cfg.Database.Type == "postgresql" {
		if cfg.Database.Host == "" || cfg.Database.Name == "" {
			return fmt.Errorf("postgresql requires host and name to be configured")
		}
	}

	if cfg.Git.Enabled && cfg.Git.AuthMethod == "ssh" && cfg.Git.SSHKeyPath == "" {
		return fmt.Errorf("git SSH auth requires ssh_key_path to be configured")
	}

	return nil
}

// Logger creates a configured zap logger based on the logging config.
func (c *Config) Logger() (*zap.Logger, error) {
	var config zap.Config

	switch c.Logging.Format {
	case "json":
		config = zap.NewProductionConfig()
	default:
		config = zap.NewDevelopmentConfig()
	}

	switch c.Logging.Level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	if c.Logging.Output == "file" && c.Logging.File != "" {
		config.OutputPaths = []string{c.Logging.File}
		config.ErrorOutputPaths = []string{c.Logging.File}
	}

	return config.Build()
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}