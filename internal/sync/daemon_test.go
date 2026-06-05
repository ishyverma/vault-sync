package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNoteEvent(t *testing.T) {
	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{"create .md", fsnotify.Event{Name: "test.md", Op: fsnotify.Create}, true},
		{"write .md", fsnotify.Event{Name: "note.md", Op: fsnotify.Write}, true},
		{"remove .md", fsnotify.Event{Name: "note.md", Op: fsnotify.Remove}, false},
		{"rename .md", fsnotify.Event{Name: "note.md", Op: fsnotify.Rename}, false},
		{"chmod .md", fsnotify.Event{Name: "note.md", Op: fsnotify.Chmod}, false},
		{"create .db", fsnotify.Event{Name: "vault.db", Op: fsnotify.Create}, false},
		{"create .toml", fsnotify.Event{Name: "config.toml", Op: fsnotify.Create}, false},
		{"no extension", fsnotify.Event{Name: "README", Op: fsnotify.Write}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNoteEvent(tt.event))
		})
	}
}

func TestDaemon_PIDLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	notesDir := filepath.Join(tmpDir, "notes")
	os.MkdirAll(notesDir, 0o755)

	store := storage.NewNoteStore(tmpDir)
	require.NoError(t, store.Init())
	engine := NewEngine(store, notesDir)
	daemon := NewDaemon(engine, notesDir, time.Minute)
	daemon.pidPath = filepath.Join(tmpDir, "vaultd.pid")

	err := daemon.writePID()
	require.NoError(t, err)

	pid, err := daemon.readPID()
	require.NoError(t, err)
	assert.Equal(t, os.Getpid(), pid)

	daemon.removePID()
	_, err = daemon.readPID()
	assert.Error(t, err)
	assert.False(t, fileExists(daemon.pidPath))
}

func TestDaemon_StatusWhenNotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	notesDir := filepath.Join(tmpDir, "notes")

	store := storage.NewNoteStore(tmpDir)
	store.Init()
	engine := NewEngine(store, notesDir)
	daemon := NewDaemon(engine, notesDir, time.Minute)
	daemon.pidPath = filepath.Join(tmpDir, "vaultd.pid")

	running, pid, err := daemon.Status()
	assert.NoError(t, err)
	assert.False(t, running)
	assert.Equal(t, 0, pid)
}

func TestDaemon_StatusWithStalePID(t *testing.T) {
	tmpDir := t.TempDir()
	notesDir := filepath.Join(tmpDir, "notes")

	store := storage.NewNoteStore(tmpDir)
	store.Init()
	engine := NewEngine(store, notesDir)
	daemon := NewDaemon(engine, notesDir, time.Minute)
	daemon.pidPath = filepath.Join(tmpDir, "vaultd.pid")

	os.WriteFile(daemon.pidPath, []byte("999999"), 0o644)

	running, pid, err := daemon.Status()
	assert.NoError(t, err)
	assert.False(t, running)
	assert.Equal(t, 0, pid)
	assert.False(t, fileExists(daemon.pidPath), "stale PID file should be removed")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
