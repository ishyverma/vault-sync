package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConnector struct {
	name     string
	pushCall int
	pullCall int
	pushErr  error
	pullErr  error
	remoteID string
	content  string
}

func (m *mockConnector) Name() string          { return m.name }
func (m *mockConnector) Connect() error        { return nil }
func (m *mockConnector) Status() (bool, error) { return true, nil }
func (m *mockConnector) Push(note *storage.Note, content string, remoteID string) (string, error) {
	m.pushCall++
	if m.pushErr != nil {
		return "", m.pushErr
	}
	m.content = content
	m.remoteID = "remote/" + note.Filename
	return m.remoteID, nil
}
func (m *mockConnector) Pull(remoteID string) (string, error) {
	m.pullCall++
	if m.pullErr != nil {
		return "", m.pullErr
	}
	return m.content, nil
}
func (m *mockConnector) Delete(remoteID string) error { return nil }

func newTestEngine(t *testing.T) (*Engine, *storage.NoteStore, string) {
	t.Helper()
	vaultDir := t.TempDir()
	store := storage.NewNoteStore(vaultDir)
	require.NoError(t, store.Init())

	notesDir := filepath.Join(vaultDir, "notes")
	require.NoError(t, os.MkdirAll(notesDir, 0o755))

	engine := NewEngine(store, notesDir)
	mock := &mockConnector{name: "obsidian"}
	engine.RegisterConnector("obsidian", mock)

	return engine, store, notesDir
}

func createTestNote(t *testing.T, store *storage.NoteStore, notesDir, id, filename, content string) {
	t.Helper()
	store.CreateNote(&storage.Note{
		ID:          id,
		Filename:    filename,
		Title:       filename,
		Path:        filename,
		ContentHash: computeHash(content),
	})
	os.WriteFile(filepath.Join(notesDir, filename), []byte(content), 0o644)
}

func TestPushNote(t *testing.T) {
	engine, store, notesDir := newTestEngine(t)
	createTestNote(t, store, notesDir, "n1", "test.md", "---\ntitle: Test\n---\n\nHello")

	err := engine.PushNote("n1")
	require.NoError(t, err)

	mock := engine.connectors["obsidian"].(*mockConnector)
	assert.Equal(t, 1, mock.pushCall)
	assert.Equal(t, "remote/test.md", mock.remoteID)

	state, err := store.GetSyncState("n1", "obsidian")
	require.NoError(t, err)
	assert.Equal(t, "synced", state.Status)
	assert.NotEmpty(t, state.LastHash)
	assert.NotEmpty(t, state.RemoteID)
}

func TestPushNote_SkipsAlreadySynced(t *testing.T) {
	engine, store, notesDir := newTestEngine(t)
	createTestNote(t, store, notesDir, "n1", "test.md", "same content")

	err := engine.PushNote("n1")
	require.NoError(t, err)

	mock := engine.connectors["obsidian"].(*mockConnector)
	firstCalls := mock.pushCall

	err = engine.PushNote("n1")
	require.NoError(t, err)

	assert.Equal(t, firstCalls, mock.pushCall, "should not push again when hash matches")
}

func TestPushNote_RePushesOnContentChange(t *testing.T) {
	engine, store, notesDir := newTestEngine(t)
	createTestNote(t, store, notesDir, "n1", "test.md", "version 1")

	err := engine.PushNote("n1")
	require.NoError(t, err)

	createTestNote(t, store, notesDir, "n1", "test.md", "version 2 — changed")

	err = engine.PushNote("n1")
	require.NoError(t, err)

	mock := engine.connectors["obsidian"].(*mockConnector)
	assert.Equal(t, 2, mock.pushCall)
}

func TestPushNote_NotFound(t *testing.T) {
	engine, _, _ := newTestEngine(t)
	err := engine.PushNote("nonexistent")
	assert.Error(t, err)
}

func TestSyncAll(t *testing.T) {
	engine, store, notesDir := newTestEngine(t)
	createTestNote(t, store, notesDir, "n1", "a.md", "note a")
	createTestNote(t, store, notesDir, "n2", "b.md", "note b")
	createTestNote(t, store, notesDir, "n3", "c.md", "note c")

	err := engine.SyncAll()
	require.NoError(t, err)

	for _, id := range []string{"n1", "n2", "n3"} {
		state, err := store.GetSyncState(id, "obsidian")
		require.NoError(t, err)
		assert.Equal(t, "synced", state.Status)
	}
}

func TestSyncStatus_NoBackends(t *testing.T) {
	store := storage.NewNoteStore(t.TempDir())
	require.NoError(t, store.Init())
	engine := NewEngine(store, t.TempDir())

	states, err := engine.SyncStatus("n1")
	require.NoError(t, err)
	assert.Len(t, states, 0)
}

func TestSyncStatus_AfterPush(t *testing.T) {
	engine, store, notesDir := newTestEngine(t)
	createTestNote(t, store, notesDir, "n1", "test.md", "content")
	engine.PushNote("n1")

	states, err := engine.SyncStatus("n1")
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, "synced", states[0].Status)
	assert.Equal(t, "obsidian", states[0].Backend)
}

func TestAllSyncStatuses(t *testing.T) {
	engine, store, notesDir := newTestEngine(t)
	createTestNote(t, store, notesDir, "n1", "a.md", "a")
	createTestNote(t, store, notesDir, "n2", "b.md", "b")

	engine.PushNote("n1")
	engine.PushNote("n2")

	states, err := engine.AllSyncStatuses()
	require.NoError(t, err)
	assert.Len(t, states, 2)
}

func TestConnectorFailure(t *testing.T) {
	engine, store, notesDir := newTestEngine(t)
	failingMock := &mockConnector{name: "obsidian", pushErr: assert.AnError}
	engine.connectors["obsidian"] = failingMock

	createTestNote(t, store, notesDir, "n1", "fail.md", "content")
	err := engine.PushNote("n1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "push errors")

	state, err := store.GetSyncState("n1", "obsidian")
	require.NoError(t, err)
	assert.Equal(t, "failed", state.Status)
	assert.Contains(t, state.ErrorMsg, "assert.AnError")
}

func TestAllSyncStatuses_NoConnectors(t *testing.T) {
	store := storage.NewNoteStore(t.TempDir())
	require.NoError(t, store.Init())
	engine := NewEngine(store, t.TempDir())

	states, err := engine.AllSyncStatuses()
	require.NoError(t, err)
	assert.Nil(t, states)
}
