package storage

import "time"

type Store interface {
	Init() error
	CreateNote(note *Note) error
	GetNote(id string) (*Note, error)
	FindNoteByFilename(filename string) (*Note, error)
	UpdateNote(note *Note) error
	DeleteNote(id string) error
	ListNotes() ([]*Note, error)
	ListNotesByTag(tag string) ([]*Note, error)
	SearchNotes(query string) ([]*Note, error)
	Close() error
}

type NoteStore struct {
	vaultDir    string
	notes       map[string]*Note
	dirty       bool
}

func NewNoteStore(vaultDir string) *NoteStore {
	return &NoteStore{
		vaultDir: vaultDir,
		notes:    make(map[string]*Note),
	}
}

func (s *NoteStore) CreateNote(note *Note) error {
	if _, exists := s.notes[note.ID]; exists {
		return ErrNoteAlreadyExists
	}
	now := time.Now()
	note.CreatedAt = now
	note.ModifiedAt = now
	s.notes[note.ID] = note
	s.dirty = true
	return nil
}

func (s *NoteStore) GetNote(id string) (*Note, error) {
	note, exists := s.notes[id]
	if !exists {
		return nil, ErrNoteNotFound
	}
	return note, nil
}

func (s *NoteStore) FindNoteByFilename(filename string) (*Note, error) {
	for _, note := range s.notes {
		if note.Filename == filename {
			return note, nil
		}
	}
	return nil, ErrNoteNotFound
}

func (s *NoteStore) UpdateNote(note *Note) error {
	if _, exists := s.notes[note.ID]; !exists {
		return ErrNoteNotFound
	}
	note.ModifiedAt = time.Now()
	s.notes[note.ID] = note
	s.dirty = true
	return nil
}

func (s *NoteStore) DeleteNote(id string) error {
	if _, exists := s.notes[id]; !exists {
		return ErrNoteNotFound
	}
	delete(s.notes, id)
	s.dirty = true
	return nil
}

func (s *NoteStore) ListNotes() ([]*Note, error) {
	result := make([]*Note, 0, len(s.notes))
	for _, note := range s.notes {
		if !note.Archived {
			result = append(result, note)
		}
	}
	return result, nil
}

func (s *NoteStore) ListNotesByTag(tag string) ([]*Note, error) {
	var result []*Note
	for _, note := range s.notes {
		if note.Archived {
			continue
		}
		for _, t := range note.Tags {
			if t == tag {
				result = append(result, note)
				break
			}
		}
	}
	return result, nil
}

func (s *NoteStore) SearchNotes(query string) ([]*Note, error) {
	if query == "" {
		return s.ListNotes()
	}
	queryLower := toLower(query)
	var result []*Note
	for _, note := range s.notes {
		if note.Archived {
			continue
		}
		if containsLower(note.Title, queryLower) ||
			containsLower(note.Filename, queryLower) {
			result = append(result, note)
		}
	}
	return result, nil
}

func (s *NoteStore) Close() error {
	return nil
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func containsLower(s, substr string) bool {
	sLower := toLower(s)
	return len(sLower) >= len(substr) && contains(sLower, substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	limit := len(s) - len(substr)
	for i := 0; i <= limit; i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (s *NoteStore) Init() error {
	return nil
}
