package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_CreatesDirectories(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	origDate := os.Getenv("VAULT_TEST_DATE")
	os.Setenv("VAULT_TEST_DATE", "2026-06-05")
	t.Cleanup(func() {
		if origDate != "" {
			os.Setenv("VAULT_TEST_DATE", origDate)
		} else {
			os.Unsetenv("VAULT_TEST_DATE")
		}
	})

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(tmpHome, ".vault", "notes"))
	assert.DirExists(t, filepath.Join(tmpHome, ".vault", "templates"))
	assert.DirExists(t, filepath.Join(tmpHome, ".vault", "attachments"))
	assert.DirExists(t, filepath.Join(tmpHome, ".config", "vault"))

	cfgPath := filepath.Join(tmpHome, ".config", "vault", "config.toml")
	assert.FileExists(t, cfgPath)

	welcomePath := filepath.Join(tmpHome, ".vault", "notes", "welcome.md")
	assert.FileExists(t, welcomePath)
}

func TestInit_WelcomeNoteContent(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	origDate := os.Getenv("VAULT_TEST_DATE")
	os.Setenv("VAULT_TEST_DATE", "2026-06-05")
	t.Cleanup(func() {
		if origDate != "" {
			os.Setenv("VAULT_TEST_DATE", origDate)
		} else {
			os.Unsetenv("VAULT_TEST_DATE")
		}
	})

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	welcomePath := filepath.Join(tmpHome, ".vault", "notes", "welcome.md")
	data, err := os.ReadFile(welcomePath)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "Welcome to VaultSync")
	assert.Contains(t, content, "vault new my-note")
	assert.Contains(t, content, filepath.Join(tmpHome, ".vault", "notes"))
}

func TestInit_CreatesTemplates(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	templates := []string{"blank.md", "daily.md", "meeting.md", "project.md"}
	for _, tmpl := range templates {
		tmplPath := filepath.Join(tmpHome, ".vault", "templates", tmpl)
		assert.FileExists(t, tmplPath, "template %s should exist", tmpl)

		data, err := os.ReadFile(tmplPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "---", "template %s should have frontmatter", tmpl)
	}
}

func TestInit_ConfigFileIsValid(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	cfgPath := filepath.Join(tmpHome, ".config", "vault", "config.toml")
	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "[vault]")
	assert.Contains(t, content, "[sync]")
	assert.Contains(t, content, "[backends]")
	assert.Contains(t, content, "editor")
	assert.Contains(t, content, "path")
}

func TestInit_EditorDetection(t *testing.T) {
	editor := detectEditor()
	if editor != "" {
		assert.NotEmpty(t, editor)
		_, err := os.Stat(editor)
		if os.IsNotExist(err) {
			_, err = os.Stat("/usr/bin/" + editor)
			if os.IsNotExist(err) {
				t.Logf("editor %q not found on PATH (may be expected in CI)", editor)
			}
		}
	}
}

func TestInit_Idempotent(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	err = rootCmd.Execute()
	require.NoError(t, err, "running init twice should not error")

	welcomePath := filepath.Join(tmpHome, ".vault", "notes", "welcome.md")
	assert.FileExists(t, welcomePath)
}
