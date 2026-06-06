package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestVault(t *testing.T) string {
	t.Helper()
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

	runEditor = func(_, _ string) error { return nil }
	t.Cleanup(func() { runEditor = func(e, p string) error { return realRunEditor(e, p) } })

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	return tmpHome
}

func TestNewCommand(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "test-note", "--no-open"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	notesDir := filepath.Join(os.Getenv("HOME"), ".vault", "notes")
	notePath := filepath.Join(notesDir, "test-note.md")
	assert.FileExists(t, notePath)

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "title: \"test-note\"")
	assert.Contains(t, content, "---")
}

func TestNewCommand_WithDailyTemplate(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "today", "--template", "daily", "--no-open"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	notesDir := filepath.Join(os.Getenv("HOME"), ".vault", "notes")
	notePath := filepath.Join(notesDir, "today.md")
	assert.FileExists(t, notePath)

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "[daily]")
	assert.Contains(t, content, "Today's Focus")
}

func TestNewCommand_AlreadyExists(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "duplicate", "--no-open"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"new", "duplicate", "--no-open"})
	err = rootCmd.Execute()
	assert.Error(t, err)
}

func TestNewCommand_WithMeetingTemplate(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "standup", "--template", "meeting", "--no-open"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	notesDir := filepath.Join(os.Getenv("HOME"), ".vault", "notes")
	notePath := filepath.Join(notesDir, "standup.md")
	assert.FileExists(t, notePath)

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "[meeting]")
	assert.Contains(t, content, "Agenda")
}

func TestNewCommand_WithProjectTemplate(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "project-x", "--template", "project", "--no-open"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	notesDir := filepath.Join(os.Getenv("HOME"), ".vault", "notes")
	notePath := filepath.Join(notesDir, "project-x.md")
	assert.FileExists(t, notePath)
}

func TestNewCommand_NoArgs(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestListCommand(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "alpha", "--no-open"})
	require.NoError(t, rootCmd.Execute())

	rootCmd.SetArgs([]string{"new", "beta", "--no-open"})
	require.NoError(t, rootCmd.Execute())

	rootCmd.SetArgs([]string{"list"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestListCommand_AfterInit(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"list"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestDeleteCommand(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "delete-me", "--no-open"})
	require.NoError(t, rootCmd.Execute())

	rootCmd.SetArgs([]string{"delete", "delete-me"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	notesDir := filepath.Join(os.Getenv("HOME"), ".vault", "notes")
	notePath := filepath.Join(notesDir, "delete-me.md")
	_, err = os.Stat(notePath)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteCommand_NotFound(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"delete", "nonexistent"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestSearchCommand(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"new", "golang-notes", "--no-open"})
	require.NoError(t, rootCmd.Execute())

	rootCmd.SetArgs([]string{"new", "rust-guide", "--no-open"})
	require.NoError(t, rootCmd.Execute())

	rootCmd.SetArgs([]string{"search", "golang"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestSearchCommand_NoResults(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"search", "zzzzz"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestDailyCommand(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"daily"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	date := time.Now().Format("2006-01-02")
	notesDir := filepath.Join(os.Getenv("HOME"), ".vault", "notes")
	dailyPath := filepath.Join(notesDir, date+".md")
	assert.FileExists(t, dailyPath)

	data, err := os.ReadFile(dailyPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "Daily Note")
	assert.Contains(t, content, "[daily]")
}

func TestDailyCommand_Idempotent(t *testing.T) {
	setupTestVault(t)

	rootCmd.SetArgs([]string{"daily"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"daily"})
	err = rootCmd.Execute()
	require.NoError(t, err, "running daily twice should not error")
}

func TestOpenCommand_WithoutDbEntry(t *testing.T) {
	setupTestVault(t)

	notesDir := filepath.Join(os.Getenv("HOME"), ".vault", "notes")
	orphanContent := "---\ntitle: Orphan Note\n---\n\nBody"
	err := os.WriteFile(filepath.Join(notesDir, "orphan.md"), []byte(orphanContent), 0o644)
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"open", "orphan"})
	err = rootCmd.Execute()
	assert.NoError(t, err)
}
