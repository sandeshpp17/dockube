// Package config provides tests for Dockube configuration management.
package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, ":8080", cfg.Server.Address)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "sqlite", cfg.Database.Type)
	assert.Equal(t, "data/dockube.db", cfg.Database.Path)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.True(t, cfg.Security.CSRFEnabled)
	assert.True(t, cfg.Cache.Enabled)
	assert.Equal(t, "development", cfg.App.Environment)
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(cfg *Config) {},
			wantErr: false,
		},
		{
			name: "invalid database type",
			modify: func(cfg *Config) {
				cfg.Database.Type = "invalid"
			},
			wantErr: true,
		},
		{
			name: "postgresql without host",
			modify: func(cfg *Config) {
				cfg.Database.Type = "postgresql"
				cfg.Database.Host = ""
				cfg.Database.Name = "dockube"
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			modify: func(cfg *Config) {
				cfg.Server.Port = 99999
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := validateConfig(cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnvOverrides(t *testing.T) {
	// Save original env
	originalDB := os.Getenv("DOCKUBE_DB_PATH")
	defer func() {
		if originalDB != "" {
			os.Setenv("DOCKUBE_DB_PATH", originalDB)
		} else {
			os.Unsetenv("DOCKUBE_DB_PATH")
		}
	}()

	// Set test env
	os.Setenv("DOCKUBE_DB_PATH", "/tmp/test.db")

	// Note: Current implementation doesn't fully support env overrides
	// This test documents the expected behavior for future implementation
	cfg := DefaultConfig()
	assert.NotEqual(t, "/tmp/test.db", cfg.Database.Path) // Current behavior
}

func TestDurationValidation(t *testing.T) {
	cfg := DefaultConfig()

	// Ensure all durations are properly set
	assert.Greater(t, cfg.Server.ReadTimeout, time.Duration(0))
	assert.Greater(t, cfg.Server.WriteTimeout, time.Duration(0))
	assert.Greater(t, cfg.Server.IdleTimeout, time.Duration(0))
	assert.Greater(t, cfg.Git.CloneTimeout, time.Duration(0))
	assert.Greater(t, cfg.Cache.TTL, time.Duration(0))
}

func TestIsProduction(t *testing.T) {
	cfg := DefaultConfig()

	cfg.App.Environment = "production"
	assert.True(t, cfg.IsProduction())
	assert.False(t, cfg.IsDevelopment())

	cfg.App.Environment = "development"
	assert.False(t, cfg.IsProduction())
	assert.True(t, cfg.IsDevelopment())
}

func TestLoggerConfig(t *testing.T) {
	cfg := DefaultConfig()

	logger, err := cfg.Logger()
	require.NoError(t, err)
	require.NotNil(t, logger)
	defer logger.Sync()

	// Test that we can log without error
	logger.Info("test log message")
}