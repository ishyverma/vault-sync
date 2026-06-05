package core

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ishyverma/vault-sync/internal/storage"
)

type Manager struct {
	store    *storage.NoteStore
	vaultDir string
	tmpl     *TemplateEngine
}

func NewManager(vaultDir string, store *storage.NoteStore, tmpl *TemplateEngine) *Manager {
	return &Manager{
		store:    store,
		vaultDir: vaultDir,
		tmpl:     tmpl,
	}
}

func (m *Manager) CreateNote(name, templateName string) (*storage.Note, error) {
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}

	notePath := filepath.Join(m.NotesDir(), name)
	if _, err := os.Stat(notePath); err == nil {
		return nil, fmt.Errorf("note already exists: %s", name)
	}

	if templateName == "" {
		templateName = "blank"
	}

	title := strings.TrimSuffix(name, ".md")
	content, err := m.tmpl.Render(templateName, title)
	if err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}

	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse generated frontmatter: %w", err)
	}

	if err := os.MkdirAll(m.NotesDir(), 0o755); err != nil {
		return nil, fmt.Errorf("create notes dir: %w", err)
	}

	if err := os.WriteFile(notePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write note: %w", err)
	}

	hash := computeHash(content)
	note := &storage.Note{
		ID:          generateID(),
		Filename:    name,
		Title:       fm.Title,
		Path:        name,
		ContentHash: hash,
		WordCount:   WordCount(content),
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
		Tags:        fm.Tags,
	}

	if err := m.store.CreateNote(note); err != nil {
		os.Remove(notePath)
		return nil, fmt.Errorf("create note in store: %w", err)
	}

	return note, nil
}

func (m *Manager) OpenNote(name string) (*storage.Note, string, error) {
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}

	note, err := m.store.FindNoteByFilename(name)
	if err != nil {
		notePath := filepath.Join(m.NotesDir(), name)
		if _, statErr := os.Stat(notePath); statErr == nil {
			fm, body, parseErr := ParseFileFrontmatter(notePath)
			if parseErr != nil {
				return nil, "", parseErr
			}
			content := BuildNoteContent(fm, body)
			note = &storage.Note{
				ID:          generateID(),
				Filename:    name,
				Title:       fm.Title,
				Path:        name,
				ContentHash: computeHash(content),
				WordCount:   WordCount(content),
				CreatedAt:   time.Now(),
				ModifiedAt:  time.Now(),
				Tags:        fm.Tags,
			}
			m.store.CreateNote(note)
		} else {
			return nil, "", fmt.Errorf("note not found: %s", name)
		}
	}

	notePath := filepath.Join(m.NotesDir(), note.Filename)
	return note, notePath, nil
}

func (m *Manager) ListNotes() ([]*storage.Note, error) {
	return m.store.ListNotes()
}

func (m *Manager) DeleteNote(name string) error {
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}

	note, err := m.store.FindNoteByFilename(name)
	if err != nil {
		return fmt.Errorf("note not found: %s", name)
	}

	notePath := filepath.Join(m.NotesDir(), name)
	if err := os.Remove(notePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}

	return m.store.DeleteNote(note.ID)
}

func (m *Manager) SearchNotes(query string) ([]*storage.Note, error) {
	return m.store.SearchNotes(query)
}

func (m *Manager) NotesDir() string {
	return filepath.Join(m.vaultDir, "notes")
}

func ParseFileFrontmatter(path string) (Frontmatter, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, "", fmt.Errorf("read file: %w", err)
	}
	return ParseFrontmatter(string(data))
}

func computeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

func generateID() string {
	b := make([]byte, 16)
	urandom(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func urandom(b []byte) {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		for i := range b {
			b[i] = byte(time.Now().UnixNano() % 256)
		}
		return
	}
	defer f.Close()
	f.Read(b)
}
