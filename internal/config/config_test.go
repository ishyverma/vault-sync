package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "~/.vault/notes", cfg.Vault.Path)
	assert.Equal(t, "nvim", cfg.Vault.Editor)
	assert.True(t, cfg.Vault.AutoDaily)
	assert.True(t, cfg.Vault.WordCountInStatusbar)

	assert.True(t, cfg.Sync.AutoSync)
	assert.Equal(t, 60, cfg.Sync.SyncInterval)
	assert.Equal(t, "ask", cfg.Sync.ConflictStrategy)
	assert.Equal(t, 5, cfg.Sync.QueueRetryLimit)
	assert.Equal(t, "exponential", cfg.Sync.QueueRetryBackoff)

	assert.True(t, cfg.Backends.Notion.Enabled)
	assert.Equal(t, "both", cfg.Backends.Notion.SyncDirection)

	assert.True(t, cfg.Backends.Obsidian.Enabled)
	assert.Equal(t, "VaultSync", cfg.Backends.Obsidian.Subfolder)

	assert.False(t, cfg.Backends.Git.Enabled)

	assert.Equal(t, "dark", cfg.TUI.Theme)
	assert.Equal(t, "modified", cfg.TUI.ListSort)

	assert.True(t, cfg.Search.Fuzzy)
	assert.Equal(t, 50, cfg.Search.MaxResults)
	assert.True(t, cfg.Search.Highlight)

	assert.False(t, cfg.Notifications.SyncSuccess)
	assert.True(t, cfg.Notifications.SyncFailure)
	assert.True(t, cfg.Notifications.ConflictDetected)

	assert.Empty(t, cfg.Hooks.PreSync)
	assert.Empty(t, cfg.Hooks.PostSync)
	assert.Empty(t, cfg.Hooks.OnConflict)
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_CONFIG_HOME")
	t.Cleanup(func() {
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		}
	})

	cfg := DefaultConfig()
	cfg.Vault.Editor = "code"
	cfg.Sync.ConflictStrategy = "local_wins"
	cfg.Backends.Obsidian.Enabled = false
	cfg.Backends.Notion.SyncDirection = "push_only"

	err := Save(&cfg)
	require.NoError(t, err)

	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "code", loaded.Vault.Editor)
	assert.Equal(t, "local_wins", loaded.Sync.ConflictStrategy)
	assert.False(t, loaded.Backends.Obsidian.Enabled)
	assert.Equal(t, "push_only", loaded.Backends.Notion.SyncDirection)
}

func TestLoad_MissingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run 'vault init' first")
}

func TestConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	dir, err := ConfigDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ".config", "vault"), dir)
}

func TestVaultDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	dir, err := VaultDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ".vault"), dir)
}

func TestSave_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cfg := DefaultConfig()
	err := Save(&cfg)
	require.NoError(t, err)

	configPath, err := ConfigPath()
	require.NoError(t, err)
	assert.FileExists(t, configPath)
}

func TestSave_ConfigFileContents(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cfg := DefaultConfig()
	err := Save(&cfg)
	require.NoError(t, err)

	configPath, err := ConfigPath()
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "editor = 'nvim'")
	assert.Contains(t, content, "[vault]")
	assert.Contains(t, content, "[sync]")
	assert.Contains(t, content, "[backends]")
	assert.Contains(t, content, "[backends.notion]")
	assert.Contains(t, content, "[backends.obsidian]")
	assert.Contains(t, content, "[backends.git]")
	assert.Contains(t, content, "[tui]")
	assert.Contains(t, content, "[search]")
	assert.Contains(t, content, "[notifications]")
	assert.Contains(t, content, "[hooks]")
}
