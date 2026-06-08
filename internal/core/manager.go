package core

import (
	"crypto/rand"
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

	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse generated frontmatter: %w", err)
	}

	if err := os.MkdirAll(m.NotesDir(), 0o755); err != nil {
		return nil, fmt.Errorf("create notes dir: %w", err)
	}

	if err := os.WriteFile(notePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write note: %w", err)
	}

	wc := WordCount(body)
	if wc == 0 {
		wc = WordCount(content)
	}
	hash := ComputeHash(content)
	id, err := GenerateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	note := &storage.Note{
		ID:          id,
		Filename:    name,
		Title:       fm.Title,
		Path:        name,
		Content:     content,
		ContentHash: hash,
		WordCount:   wc,
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
			noteID, genErr := GenerateID()
			if genErr != nil {
				return nil, "", fmt.Errorf("generate id: %w", genErr)
			}
			note = &storage.Note{
				ID:          noteID,
				Filename:    name,
				Title:       fm.Title,
				Path:        name,
				Content:     content,
				ContentHash: ComputeHash(content),
				WordCount:   WordCount(content),
				CreatedAt:   time.Now(),
				ModifiedAt:  time.Now(),
				Tags:        fm.Tags,
			}
			if err := m.store.CreateNote(note); err != nil {
				return nil, "", fmt.Errorf("create note in store: %w", err)
			}
		} else {
			return nil, "", fmt.Errorf("note not found: %s", name)
		}
	}

	notePath := filepath.Join(m.NotesDir(), note.Filename)
	return note, notePath, nil
}

func (m *Manager) GetNote(noteID string) (*storage.Note, error) {
	return m.store.GetNote(noteID)
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

func (m *Manager) ListNotesByTag(tag string) ([]*storage.Note, error) {
	return m.store.ListNotesByTag(tag)
}

func (m *Manager) SearchNotes(query string) ([]*storage.Note, error) {
	return m.store.SearchNotes(query)
}

func (m *Manager) SyncFromDisk(noteID string) error {
	note, err := m.store.GetNote(noteID)
	if err != nil {
		return fmt.Errorf("get note: %w", err)
	}

	notePath := filepath.Join(m.NotesDir(), note.Filename)
	data, err := os.ReadFile(notePath)
	if err != nil {
		return fmt.Errorf("read note file: %w", err)
	}

	content := string(data)
	fm, body, parseErr := ParseFrontmatter(content)
	if parseErr != nil {
		return fmt.Errorf("parse frontmatter: %w", parseErr)
	}
	if body == "" {
		body = content
	}

	note.Title = fm.Title
	note.Tags = fm.Tags
	note.Content = content
	note.ContentHash = ComputeHash(content)
	note.WordCount = WordCount(body)

	return m.store.UpdateNote(note)
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

func ComputeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

func GenerateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
