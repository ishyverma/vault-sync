package notion

import (
	"fmt"
	"sync"
)

// Mapper handles bidirectional mapping between local note IDs and Notion page IDs.
type Mapper struct {
	mu       sync.RWMutex
	pageByID map[string]string // noteID -> pageID
	idByPage map[string]string // pageID -> noteID
}

// NewMapper creates a new mapper.
func NewMapper() *Mapper {
	return &Mapper{
		pageByID: make(map[string]string),
		idByPage: make(map[string]string),
	}
}

// Set records a mapping between a local note ID and a Notion page ID.
func (m *Mapper) Set(noteID, pageID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pageByID[noteID] = pageID
	m.idByPage[pageID] = noteID
}

// GetPageID returns the Notion page ID for a given local note ID.
func (m *Mapper) GetPageID(noteID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pageID, ok := m.pageByID[noteID]
	if !ok {
		return "", fmt.Errorf("no page mapping for note: %s", noteID)
	}
	return pageID, nil
}

// GetNoteID returns the local note ID for a given Notion page ID.
func (m *Mapper) GetNoteID(pageID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	noteID, ok := m.idByPage[pageID]
	if !ok {
		return "", fmt.Errorf("no note mapping for page: %s", pageID)
	}
	return noteID, nil
}

// Delete removes a mapping by note ID.
func (m *Mapper) Delete(noteID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pageID, ok := m.pageByID[noteID]; ok {
		delete(m.idByPage, pageID)
	}
	delete(m.pageByID, noteID)
}

// All returns all noteID->pageID mappings.
func (m *Mapper) All() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]string, len(m.pageByID))
	for k, v := range m.pageByID {
		result[k] = v
	}
	return result
}
