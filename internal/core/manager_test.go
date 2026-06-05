package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	vaultDir := t.TempDir()
	store := storage.NewNoteStore(vaultDir)
	tmpl := NewTemplateEngine()
	m := NewManager(vaultDir, store, tmpl)
	return m, vaultDir
}

func TestCreateNote(t *testing.T) {
	m, _ := newTestManager(t)
	note, err := m.CreateNote("test-note", "blank")
	require.NoError(t, err)

	assert.Equal(t, "test-note.md", note.Filename)
	assert.Equal(t, "test-note", note.Title)
	assert.NotEmpty(t, note.ID)
	assert.True(t, note.WordCount > 0)

	notePath := filepath.Join(m.NotesDir(), "test-note.md")
	_, err = os.Stat(notePath)
	assert.NoError(t, err, "note file should exist")

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "title: \"test-note\"")
	assert.Contains(t, content, "---")
}

func TestCreateNote_AlreadyExists(t *testing.T) {
	m, _ := newTestManager(t)
	_, err := m.CreateNote("exists", "blank")
	require.NoError(t, err)

	_, err = m.CreateNote("exists", "blank")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreateNote_AddsMdExtension(t *testing.T) {
	m, _ := newTestManager(t)
	note, err := m.CreateNote("my-note.md", "blank")
	require.NoError(t, err)
	assert.Equal(t, "my-note.md", note.Filename)
}

func TestCreateNote_DailyTemplate(t *testing.T) {
	m, _ := newTestManager(t)
	note, err := m.CreateNote("today", "daily")
	require.NoError(t, err)
	assert.Contains(t, note.Tags, "daily")
}

func TestCreateNote_MeetingTemplate(t *testing.T) {
	m, _ := newTestManager(t)
	note, err := m.CreateNote("standup", "meeting")
	require.NoError(t, err)
	assert.Contains(t, note.Tags, "meeting")
}

func TestCreateNote_ProjectTemplate(t *testing.T) {
	m, _ := newTestManager(t)
	note, err := m.CreateNote("my-project", "project")
	require.NoError(t, err)
	assert.Contains(t, note.Tags, "project")
}

func TestOpenNote(t *testing.T) {
	m, _ := newTestManager(t)
	created, err := m.CreateNote("open-me", "blank")
	require.NoError(t, err)

	note, path, err := m.OpenNote("open-me")
	require.NoError(t, err)
	assert.Equal(t, created.ID, note.ID)
	assert.FileExists(t, path)
}

func TestOpenNote_WithoutDbEntry(t *testing.T) {
	m, vaultDir := newTestManager(t)
	notesDir := filepath.Join(vaultDir, "notes")
	os.MkdirAll(notesDir, 0o755)
	content := "---\ntitle: Orphan\n---\n\nBody text"
	os.WriteFile(filepath.Join(notesDir, "orphan.md"), []byte(content), 0o644)

	note, path, err := m.OpenNote("orphan")
	require.NoError(t, err)
	assert.Equal(t, "orphan.md", note.Filename)
	assert.Equal(t, "Orphan", note.Title)
	assert.FileExists(t, path)
}

func TestOpenNote_NotFound(t *testing.T) {
	m, _ := newTestManager(t)
	_, _, err := m.OpenNote("nope")
	assert.Error(t, err)
}

func TestDeleteNote(t *testing.T) {
	m, _ := newTestManager(t)
	_, err := m.CreateNote("delete-me", "blank")
	require.NoError(t, err)

	err = m.DeleteNote("delete-me")
	require.NoError(t, err)

	notePath := filepath.Join(m.NotesDir(), "delete-me.md")
	_, err = os.Stat(notePath)
	assert.True(t, os.IsNotExist(err))

	notes, _ := m.ListNotes()
	assert.Len(t, notes, 0)
}

func TestDeleteNote_NotFound(t *testing.T) {
	m, _ := newTestManager(t)
	err := m.DeleteNote("ghost")
	assert.Error(t, err)
}

func TestListNotes(t *testing.T) {
	m, _ := newTestManager(t)
	m.CreateNote("first", "blank")
	m.CreateNote("second", "daily")
	m.CreateNote("third", "meeting")

	notes, err := m.ListNotes()
	require.NoError(t, err)
	assert.Len(t, notes, 3)
}

func TestSearchNotes(t *testing.T) {
	m, _ := newTestManager(t)
	m.CreateNote("golang-notes", "blank")
	m.CreateNote("rust-guide", "blank")
	m.CreateNote("project-plan", "blank")

	t.Run("by title", func(t *testing.T) {
		notes, err := m.SearchNotes("golang")
		require.NoError(t, err)
		assert.Len(t, notes, 1)
	})

	t.Run("case insensitive", func(t *testing.T) {
		notes, err := m.SearchNotes("RUST")
		require.NoError(t, err)
		assert.Len(t, notes, 1)
	})
}

func TestCreateNote_WithTags(t *testing.T) {
	m, _ := newTestManager(t)
	note, err := m.CreateNote("tagged", "blank")
	require.NoError(t, err)
	assert.NotNil(t, note.Tags)
}

func TestManager_NotesDir(t *testing.T) {
	m, vaultDir := newTestManager(t)
	assert.Equal(t, filepath.Join(vaultDir, "notes"), m.NotesDir())
}
