package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncState_CreateAndGet(t *testing.T) {
	s := newTestStore(t)
	state := &SyncState{
		NoteID:   "test-note",
		Backend:  "obsidian",
		RemoteID: "test/path.md",
		Status:   "local_only",
	}
	err := s.UpsertSyncState(state)
	require.NoError(t, err)

	got, err := s.GetSyncState("test-note", "obsidian")
	require.NoError(t, err)
	assert.Equal(t, "test-note", got.NoteID)
	assert.Equal(t, "obsidian", got.Backend)
	assert.Equal(t, "test/path.md", got.RemoteID)
	assert.Equal(t, "local_only", got.Status)
}

func TestSyncState_Update(t *testing.T) {
	s := newTestStore(t)
	err := s.UpsertSyncState(&SyncState{NoteID: "n1", Backend: "obsidian", Status: "local_only"})
	require.NoError(t, err)

	err = s.UpsertSyncState(&SyncState{NoteID: "n1", Backend: "obsidian", Status: "synced", LastHash: "abc123"})
	require.NoError(t, err)

	got, err := s.GetSyncState("n1", "obsidian")
	require.NoError(t, err)
	assert.Equal(t, "synced", got.Status)
	assert.Equal(t, "abc123", got.LastHash)
}

func TestSyncState_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetSyncState("nope", "obsidian")
	assert.ErrorIs(t, err, ErrSyncStateNotFound)
}

func TestSyncState_Delete(t *testing.T) {
	s := newTestStore(t)
	err := s.UpsertSyncState(&SyncState{NoteID: "n1", Backend: "obsidian", Status: "synced"})
	require.NoError(t, err)

	err = s.DeleteSyncState("n1", "obsidian")
	require.NoError(t, err)

	_, err = s.GetSyncState("n1", "obsidian")
	assert.ErrorIs(t, err, ErrSyncStateNotFound)
}

func TestSyncState_List(t *testing.T) {
	s := newTestStore(t)
	s.UpsertSyncState(&SyncState{NoteID: "n1", Backend: "obsidian", Status: "synced"})
	s.UpsertSyncState(&SyncState{NoteID: "n2", Backend: "obsidian", Status: "local_only"})
	s.UpsertSyncState(&SyncState{NoteID: "n3", Backend: "notion", Status: "synced"})

	all, err := s.ListSyncStates()
	require.NoError(t, err)
	assert.Len(t, all, 3)

	synced, err := s.ListSyncStatesByStatus("synced")
	require.NoError(t, err)
	assert.Len(t, synced, 2)

	local, err := s.ListSyncStatesByStatus("local_only")
	require.NoError(t, err)
	assert.Len(t, local, 1)
}

func TestSyncQueue_EnqueueDequeue(t *testing.T) {
	s := newTestStore(t)
	err := s.EnqueueSyncJob("n1", []string{"obsidian"}, "push")
	require.NoError(t, err)

	length, err := s.QueueLength()
	require.NoError(t, err)
	assert.Equal(t, 1, length)

	item, err := s.DequeueSyncJob()
	require.NoError(t, err)
	assert.Equal(t, "n1", item.NoteID)
	assert.Equal(t, "push", item.Direction)
	assert.Equal(t, []string{"obsidian"}, item.Backends)

	length, err = s.QueueLength()
	require.NoError(t, err)
	assert.Equal(t, 0, length)
}

func TestSyncQueue_Empty(t *testing.T) {
	s := newTestStore(t)
	_, err := s.DequeueSyncJob()
	assert.ErrorIs(t, err, ErrSyncJobNotFound)
}

func TestSyncQueue_Multiple(t *testing.T) {
	s := newTestStore(t)
	s.EnqueueSyncJob("n1", []string{"obsidian"}, "push")
	s.EnqueueSyncJob("n2", []string{"obsidian", "notion"}, "push")
	s.EnqueueSyncJob("n3", []string{"notion"}, "pull")

	length, err := s.QueueLength()
	require.NoError(t, err)
	assert.Equal(t, 3, length)

	first, _ := s.DequeueSyncJob()
	assert.Equal(t, "n1", first.NoteID)

	second, _ := s.DequeueSyncJob()
	assert.Equal(t, "n2", second.NoteID)

	third, _ := s.DequeueSyncJob()
	assert.Equal(t, "n3", third.NoteID)

	length, _ = s.QueueLength()
	assert.Equal(t, 0, length)
}

func TestSyncHistory_AddAndList(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	err := s.AddSyncHistory(&SyncHistoryEntry{
		NoteID:    "n1",
		Backend:   "obsidian",
		Direction: "push",
		Status:    "success",
		SyncedAt:  now,
		Hash:      "abc",
	})
	require.NoError(t, err)

	err = s.AddSyncHistory(&SyncHistoryEntry{
		NoteID:    "n1",
		Backend:   "notion",
		Direction: "push",
		Status:    "failed",
		SyncedAt:  now,
	})
	require.NoError(t, err)

	entries, err := s.ListSyncHistory("n1")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	backends := make(map[string]bool)
	for _, e := range entries {
		backends[e.Backend] = true
	}
	assert.True(t, backends["obsidian"])
	assert.True(t, backends["notion"])
}

func TestSyncHistory_Empty(t *testing.T) {
	s := newTestStore(t)
	entries, err := s.ListSyncHistory("nonexistent")
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}
