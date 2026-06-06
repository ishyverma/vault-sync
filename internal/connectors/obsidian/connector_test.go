package obsidian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConnector(t *testing.T) (*Connector, string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "Obsidian", "MyVault")
	notesDir := filepath.Join(tmpDir, ".vault", "notes")

	c := NewConnector(vaultPath, "VaultSync", notesDir, false)
	err := c.Connect()
	require.NoError(t, err)

	return c, tmpDir, vaultPath
}

func TestName(t *testing.T) {
	c := NewConnector("/tmp", "test", "/tmp/notes", false)
	assert.Equal(t, "obsidian", c.Name())
}

func TestConnect_CreatesTargetDir(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "Obsidian")
	c := NewConnector(vaultPath, "VaultSync", filepath.Join(tmpDir, "notes"), false)

	err := c.Connect()
	require.NoError(t, err)

	targetDir := filepath.Join(vaultPath, "VaultSync")
	info, err := os.Stat(targetDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestStatus_Healthy(t *testing.T) {
	c, _, vaultPath := newTestConnector(t)
	healthy, err := c.Status()
	assert.True(t, healthy)
	assert.NoError(t, err)
	_ = vaultPath
}

func TestStatus_Unhealthy(t *testing.T) {
	c := NewConnector("/nonexistent/path/xyz123", "VaultSync", "/tmp/notes", false)
	healthy, err := c.Status()
	assert.False(t, healthy)
	assert.Error(t, err)
}

func TestPush_CreatesFile(t *testing.T) {
	c, _, _ := newTestConnector(t)
	note := &storage.Note{
		ID:       "test-1",
		Filename: "my-note.md",
		Path:     "my-note.md",
		Title:    "My Note",
	}

	remoteID, err := c.Push(note, "---\ntitle: My Note\n---\n\nBody content", "")
	require.NoError(t, err)

	expectedPath := filepath.Join(c.targetDir, "my-note.md")
	assert.Equal(t, expectedPath, remoteID)
	assert.FileExists(t, expectedPath)

	data, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Body content")
}

func TestPush_WithSubfolder(t *testing.T) {
	c, _, _ := newTestConnector(t)
	note := &storage.Note{
		ID:       "test-2",
		Filename: "meeting.md",
		Path:     "work/meeting.md",
		Folder:   "work",
	}

	remoteID, err := c.Push(note, "---\ntitle: Meeting\n---\n\nNotes", "")
	require.NoError(t, err)

	expectedPath := filepath.Join(c.targetDir, "work", "meeting.md")
	assert.Equal(t, expectedPath, remoteID)
	assert.FileExists(t, expectedPath)
}

func TestPush_OverwritesExisting(t *testing.T) {
	c, _, _ := newTestConnector(t)
	note := &storage.Note{ID: "test-3", Filename: "update.md", Path: "update.md"}

	_, err := c.Push(note, "---\ntitle: V1\n---\n\nFirst version", "")
	require.NoError(t, err)

	_, err = c.Push(note, "---\ntitle: V2\n---\n\nSecond version", "")
	require.NoError(t, err)

	expectedPath := filepath.Join(c.targetDir, "update.md")
	data, _ := os.ReadFile(expectedPath)
	assert.Contains(t, string(data), "Second version")
}

func TestPull_ReturnsContent(t *testing.T) {
	c, _, _ := newTestConnector(t)
	note := &storage.Note{ID: "test-4", Filename: "pull-test.md", Path: "pull-test.md"}
	content := "---\ntitle: Pull Test\n---\n\nPull body"

	remoteID, err := c.Push(note, content, "")
	require.NoError(t, err)

	got, err := c.Pull(remoteID)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestPull_NotFound(t *testing.T) {
	c, _, _ := newTestConnector(t)
	_, err := c.Pull("/nonexistent/file.md")
	assert.Error(t, err)
}

func TestDelete_RemovesFile(t *testing.T) {
	c, _, _ := newTestConnector(t)
	note := &storage.Note{ID: "test-5", Filename: "delete-me.md", Path: "delete-me.md"}
	content := "---\ntitle: Delete Me\n---"

	remoteID, err := c.Push(note, content, "")
	require.NoError(t, err)
	assert.FileExists(t, remoteID)

	err = c.Delete(remoteID)
	require.NoError(t, err)
	assert.NoFileExists(t, remoteID)
}

func TestDelete_NotExists(t *testing.T) {
	c, _, _ := newTestConnector(t)
	err := c.Delete("/nonexistent/path.md")
	assert.NoError(t, err)
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := expandHome("~/Documents/Obsidian")
	expected := filepath.Join(home, "Documents/Obsidian")
	assert.Equal(t, expected, result)

	result = expandHome("/absolute/path")
	assert.Equal(t, "/absolute/path", result)
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.md")

	err := atomicWrite(path, "hello world")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}
