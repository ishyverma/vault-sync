package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *NoteStore {
	t.Helper()
	tmpDir := t.TempDir()
	s := NewNoteStore(tmpDir)
	err := s.Init()
	require.NoError(t, err)
	return s
}

func TestCreateAndGetNote(t *testing.T) {
	s := newTestStore(t)
	note := &Note{
		ID:       "test-1",
		Filename: "my-note.md",
		Title:    "My Note",
		Tags:     []string{"dev", "go"},
	}
	err := s.CreateNote(note)
	require.NoError(t, err)

	got, err := s.GetNote("test-1")
	require.NoError(t, err)
	assert.Equal(t, "my-note.md", got.Filename)
	assert.Equal(t, "My Note", got.Title)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.ModifiedAt.IsZero())
	assert.Equal(t, []string{"dev", "go"}, got.Tags)
}

func TestCreateNote_MissingID(t *testing.T) {
	s := newTestStore(t)
	err := s.CreateNote(&Note{Filename: "no-id.md"})
	assert.ErrorIs(t, err, ErrNoteIDRequired)
}

func TestCreateAndGetNote_WithFolder(t *testing.T) {
	s := newTestStore(t)
	note := &Note{
		ID:       "folder-note",
		Filename: "work/meeting.md",
		Title:    "Meeting",
		Folder:   "work",
	}
	err := s.CreateNote(note)
	require.NoError(t, err)

	got, err := s.GetNote("folder-note")
	require.NoError(t, err)
	assert.Equal(t, "work", got.Folder)
}

func TestGetNonexistentNote(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetNote("nope")
	assert.ErrorIs(t, err, ErrNoteNotFound)
}

func TestFindNoteByFilename(t *testing.T) {
	s := newTestStore(t)
	err := s.CreateNote(&Note{ID: "1", Filename: "meeting.md"})
	require.NoError(t, err)
	err = s.CreateNote(&Note{ID: "2", Filename: "notes.md"})
	require.NoError(t, err)

	note, err := s.FindNoteByFilename("meeting.md")
	require.NoError(t, err)
	assert.Equal(t, "1", note.ID)

	_, err = s.FindNoteByFilename("nope.md")
	assert.ErrorIs(t, err, ErrNoteNotFound)
}

func TestUpdateNote(t *testing.T) {
	s := newTestStore(t)
	err := s.CreateNote(&Note{ID: "u1", Filename: "u1.md", Title: "Old"})
	require.NoError(t, err)

	note, _ := s.GetNote("u1")
	note.Title = "Updated"
	err = s.UpdateNote(note)
	require.NoError(t, err)

	got, _ := s.GetNote("u1")
	assert.Equal(t, "Updated", got.Title)
}

func TestUpdateNote_UpdatesTags(t *testing.T) {
	s := newTestStore(t)
	err := s.CreateNote(&Note{ID: "tags-update", Filename: "tags.md", Title: "Tags", Tags: []string{"a", "b"}})
	require.NoError(t, err)

	note, _ := s.GetNote("tags-update")
	note.Tags = []string{"c", "d"}
	err = s.UpdateNote(note)
	require.NoError(t, err)

	got, _ := s.GetNote("tags-update")
	assert.Equal(t, []string{"c", "d"}, got.Tags)
}

func TestUpdateNonexistentNote(t *testing.T) {
	s := newTestStore(t)
	err := s.UpdateNote(&Note{ID: "ghost"})
	assert.ErrorIs(t, err, ErrNoteNotFound)
}

func TestDeleteNote(t *testing.T) {
	s := newTestStore(t)
	err := s.CreateNote(&Note{ID: "del", Filename: "del.md", Tags: []string{"gone"}})
	require.NoError(t, err)

	err = s.DeleteNote("del")
	require.NoError(t, err)

	_, err = s.GetNote("del")
	assert.ErrorIs(t, err, ErrNoteNotFound)
}

func TestListNotes(t *testing.T) {
	s := newTestStore(t)
	s.CreateNote(&Note{ID: "a", Filename: "a.md", Title: "A"})
	s.CreateNote(&Note{ID: "b", Filename: "b.md", Title: "B"})
	s.CreateNote(&Note{ID: "c", Filename: "c.md", Title: "C", Archived: true})

	notes, err := s.ListNotes()
	require.NoError(t, err)
	assert.Len(t, notes, 2)
}

func TestListNotesByTag(t *testing.T) {
	s := newTestStore(t)
	s.CreateNote(&Note{ID: "1", Filename: "a.md", Tags: []string{"go"}})
	s.CreateNote(&Note{ID: "2", Filename: "b.md", Tags: []string{"rust"}})
	s.CreateNote(&Note{ID: "3", Filename: "c.md", Tags: []string{"go", "test"}})

	notes, err := s.ListNotesByTag("go")
	require.NoError(t, err)
	assert.Len(t, notes, 2)
}

func TestSearchNotes(t *testing.T) {
	s := newTestStore(t)
	s.CreateNote(&Note{ID: "1", Title: "Rust Learning", Filename: "rust.md"})
	s.CreateNote(&Note{ID: "2", Title: "Go Notes", Filename: "go.md"})
	s.CreateNote(&Note{ID: "3", Title: "Project Plan", Filename: "plan.md"})

	t.Run("match by FTS", func(t *testing.T) {
		notes, err := s.SearchNotes("rust")
		require.NoError(t, err)
		assert.Len(t, notes, 1)
	})

	t.Run("empty query returns all", func(t *testing.T) {
		notes, err := s.SearchNotes("")
		require.NoError(t, err)
		assert.Len(t, notes, 3)
	})

	t.Run("no match", func(t *testing.T) {
		notes, err := s.SearchNotes("zzzzz")
		require.NoError(t, err)
		assert.Len(t, notes, 0)
	})
}

func TestNoteTimestamps(t *testing.T) {
	s := newTestStore(t)
	before := time.Now()
	note := &Note{ID: "ts", Filename: "ts.md"}
	err := s.CreateNote(note)
	require.NoError(t, err)
	after := time.Now()

	assert.False(t, note.CreatedAt.Before(before.Add(-time.Second)))
	assert.False(t, note.CreatedAt.After(after.Add(time.Second)))
	assert.Equal(t, note.CreatedAt, note.ModifiedAt)

	time.Sleep(time.Millisecond * 10)
	note.Title = "updated"
	s.UpdateNote(note)
	assert.True(t, note.ModifiedAt.After(note.CreatedAt))
}

func TestInit_CreatesDBFile(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewNoteStore(tmpDir)
	err := s.Init()
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "vault.db")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "vault.db should exist")
}

func TestInit_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewNoteStore(tmpDir)
	err := s.Init()
	require.NoError(t, err)
	err = s.Init()
	require.NoError(t, err, "Init should be idempotent")
}

func TestClose(t *testing.T) {
	s := newTestStore(t)
	err := s.Close()
	assert.NoError(t, err)
}

func TestPersistAcrossStores(t *testing.T) {
	tmpDir := t.TempDir()
	s1 := NewNoteStore(tmpDir)
	err := s1.Init()
	require.NoError(t, err)

	err = s1.CreateNote(&Note{ID: "persist", Filename: "persist.md", Title: "Persisted"})
	require.NoError(t, err)
	s1.Close()

	s2 := NewNoteStore(tmpDir)
	err = s2.Init()
	require.NoError(t, err)

	note, err := s2.GetNote("persist")
	require.NoError(t, err)
	assert.Equal(t, "Persisted", note.Title)
	s2.Close()
}
