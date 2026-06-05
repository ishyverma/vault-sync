package sync

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ishyverma/vault-sync/internal/connectors"
	"github.com/ishyverma/vault-sync/internal/storage"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusSynced
	StatusPending
	StatusConflict
	StatusFailed
	StatusLocalOnly
)

func (s Status) String() string {
	switch s {
	case StatusSynced:
		return "synced"
	case StatusPending:
		return "pending"
	case StatusConflict:
		return "conflict"
	case StatusFailed:
		return "failed"
	case StatusLocalOnly:
		return "local_only"
	default:
		return "unknown"
	}
}

type Engine struct {
	store      *storage.NoteStore
	connectors map[string]connectors.Connector
	notesDir   string
}

func NewEngine(store *storage.NoteStore, notesDir string) *Engine {
	return &Engine{
		store:      store,
		connectors: make(map[string]connectors.Connector),
		notesDir:   notesDir,
	}
}

func (e *Engine) RegisterConnector(name string, c connectors.Connector) {
	e.connectors[name] = c
}

func (e *Engine) PushNote(noteID string) error {
	note, err := e.store.GetNote(noteID)
	if err != nil {
		return fmt.Errorf("get note: %w", err)
	}

	content, err := e.readNoteFile(note.Filename)
	if err != nil {
		return fmt.Errorf("read note file: %w", err)
	}

	currentHash := computeHash(content)

	for name, conn := range e.connectors {
		state, stateErr := e.store.GetSyncState(noteID, name)
		if stateErr == nil && state.LastHash == currentHash && state.Status == "synced" {
			continue
		}

		if err := conn.Connect(); err != nil {
			e.recordFailure(noteID, name, err)
			continue
		}

		if stateErr == nil && state.Status == "synced" {
			conflict, checkErr := e.detectConflict(conn, state)
			if checkErr == nil && conflict {
				e.store.UpsertSyncState(&storage.SyncState{
					NoteID: noteID,
					Backend: name,
					Status: "conflict",
					ErrorMsg: "remote file modified externally",
				})
				continue
			}
		}

		remoteID, pushErr := conn.Push(note, content)
		if pushErr != nil {
			e.recordFailure(noteID, name, pushErr)
			continue
		}

		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    noteID,
			Backend:   name,
			Direction: "push",
			Status:    "success",
			SyncedAt:  time.Now().UTC(),
			Hash:      currentHash,
		})

		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     noteID,
			Backend:    name,
			RemoteID:   remoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   currentHash,
			Status:     "synced",
		})
	}

	return nil
}

func (e *Engine) SyncAll() error {
	notes, err := e.store.ListNotes()
	if err != nil {
		return fmt.Errorf("list notes: %w", err)
	}

	for _, note := range notes {
		if err := e.PushNote(note.ID); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) SyncStatus(noteID string) ([]*storage.SyncState, error) {
	var states []*storage.SyncState

	if len(e.connectors) == 0 {
		return states, nil
	}

	for name := range e.connectors {
		state, err := e.store.GetSyncState(noteID, name)
		if err != nil {
			states = append(states, &storage.SyncState{
				NoteID:  noteID,
				Backend: name,
				Status:  "local_only",
			})
			continue
		}
		states = append(states, state)
	}

	return states, nil
}

func (e *Engine) AllSyncStatuses() ([]*storage.SyncState, error) {
	if len(e.connectors) == 0 {
		return nil, nil
	}

	states, err := e.store.ListSyncStates()
	if err != nil {
		return nil, err
	}

	connectorNames := make(map[string]bool)
	for name := range e.connectors {
		connectorNames[name] = true
	}

	var result []*storage.SyncState
	for _, s := range states {
		if connectorNames[s.Backend] {
			result = append(result, s)
		}
	}

	return result, nil
}

func (e *Engine) detectConflict(conn connectors.Connector, state *storage.SyncState) (bool, error) {
	content, err := conn.Pull(state.RemoteID)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	remoteHash := computeHash(content)
	return remoteHash != state.LastHash, nil
}

func (e *Engine) recordFailure(noteID, backend string, err error) {
	e.store.AddSyncHistory(&storage.SyncHistoryEntry{
		NoteID:    noteID,
		Backend:   backend,
		Direction: "push",
		Status:    "failed",
		SyncedAt:  time.Now().UTC(),
	})

	e.store.UpsertSyncState(&storage.SyncState{
		NoteID:   noteID,
		Backend:  backend,
		Status:   "failed",
		ErrorMsg: err.Error(),
	})
}

func (e *Engine) readNoteFile(filename string) (string, error) {
	path := filepath.Join(e.notesDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func computeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}
