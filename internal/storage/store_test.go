package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetNote(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
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

func TestCreateDuplicateNote(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
	err := s.CreateNote(&Note{ID: "dup"})
	require.NoError(t, err)
	err = s.CreateNote(&Note{ID: "dup"})
	assert.ErrorIs(t, err, ErrNoteAlreadyExists)
}

func TestGetNonexistentNote(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
	_, err := s.GetNote("nope")
	assert.ErrorIs(t, err, ErrNoteNotFound)
}

func TestFindNoteByFilename(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
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
	s := NewNoteStore("/tmp/test-vault")
	err := s.CreateNote(&Note{ID: "u1", Title: "Old"})
	require.NoError(t, err)

	note, _ := s.GetNote("u1")
	note.Title = "Updated"
	err = s.UpdateNote(note)
	require.NoError(t, err)

	got, _ := s.GetNote("u1")
	assert.Equal(t, "Updated", got.Title)
}

func TestUpdateNonexistentNote(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
	err := s.UpdateNote(&Note{ID: "ghost"})
	assert.ErrorIs(t, err, ErrNoteNotFound)
}

func TestDeleteNote(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
	err := s.CreateNote(&Note{ID: "del"})
	require.NoError(t, err)

	err = s.DeleteNote("del")
	require.NoError(t, err)

	_, err = s.GetNote("del")
	assert.ErrorIs(t, err, ErrNoteNotFound)
}

func TestListNotes(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
	s.CreateNote(&Note{ID: "a", Filename: "a.md"})
	s.CreateNote(&Note{ID: "b", Filename: "b.md"})
	s.CreateNote(&Note{ID: "c", Filename: "c.md", Archived: true})

	notes, err := s.ListNotes()
	require.NoError(t, err)
	assert.Len(t, notes, 2)
}

func TestListNotesByTag(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
	s.CreateNote(&Note{ID: "1", Tags: []string{"go"}})
	s.CreateNote(&Note{ID: "2", Tags: []string{"rust"}})
	s.CreateNote(&Note{ID: "3", Tags: []string{"go", "test"}})

	notes, err := s.ListNotesByTag("go")
	require.NoError(t, err)
	assert.Len(t, notes, 2)
}

func TestSearchNotes(t *testing.T) {
	s := NewNoteStore("/tmp/test-vault")
	s.CreateNote(&Note{ID: "1", Title: "Rust Learning", Filename: "rust.md"})
	s.CreateNote(&Note{ID: "2", Title: "Go Notes", Filename: "go.md"})
	s.CreateNote(&Note{ID: "3", Title: "Project Plan", Filename: "plan.md"})

	t.Run("match title", func(t *testing.T) {
		notes, err := s.SearchNotes("rust")
		require.NoError(t, err)
		assert.Len(t, notes, 1)
	})

	t.Run("match filename", func(t *testing.T) {
		notes, err := s.SearchNotes("plan")
		require.NoError(t, err)
		assert.Len(t, notes, 1)
	})

	t.Run("case insensitive", func(t *testing.T) {
		notes, err := s.SearchNotes("RUST")
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
	s := NewNoteStore("/tmp/test-vault")
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
