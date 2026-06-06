package sync

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ishyverma/vault-sync/internal/connectors"
	"github.com/ishyverma/vault-sync/internal/core"
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
	mu         sync.RWMutex
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
	if c == nil {
		panic("sync: RegisterConnector called with nil connector")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connectors[name] = c
}

func (e *Engine) getConnectors() map[string]connectors.Connector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	cp := make(map[string]connectors.Connector, len(e.connectors))
	for k, v := range e.connectors {
		cp[k] = v
	}
	return cp
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

	for name, conn := range e.getConnectors() {
		state, stateErr := e.store.GetSyncState(noteID, name)
		if stateErr == nil && state.LastHash == currentHash && state.Status == "synced" {
			continue
		}

		if stateErr == nil && state.Status == "conflict" {
			conflict, checkErr := e.detectConflict(conn, state)
			if checkErr != nil {
				e.recordFailure(noteID, name, fmt.Errorf("conflict check: %w", checkErr))
				continue
			}
			if conflict {
				continue
			}
		}

		if err := conn.Connect(); err != nil {
			e.recordFailure(noteID, name, err)
			continue
		}

		if stateErr == nil && state.Status == "synced" {
			conflict, checkErr := e.detectConflict(conn, state)
			if checkErr != nil {
				e.recordFailure(noteID, name, fmt.Errorf("conflict check: %w", checkErr))
				continue
			}
			if conflict {
				e.store.UpsertSyncState(&storage.SyncState{
					NoteID:   noteID,
					Backend:  name,
					RemoteID: state.RemoteID,
					LastHash: state.LastHash,
					Status:   "conflict",
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

func (e *Engine) PushAll() error {
	notes, err := e.store.ListNotes()
	if err != nil {
		return fmt.Errorf("list notes: %w", err)
	}
	var firstErr error
	for _, note := range notes {
		if err := e.PushNote(note.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (e *Engine) PullNote(noteID string) error {
	note, err := e.store.GetNote(noteID)
	if err != nil {
		return fmt.Errorf("get note: %w", err)
	}

	for name, conn := range e.getConnectors() {
		state, err := e.store.GetSyncState(noteID, name)
		if err != nil || state.RemoteID == "" {
			continue
		}

		remoteContent, pullErr := conn.Pull(state.RemoteID)
		if pullErr != nil {
			if os.IsNotExist(pullErr) {
				continue
			}
			e.recordPullFailure(noteID, name, pullErr)
			continue
		}

		remoteHash := computeHash(remoteContent)
		if remoteHash == state.LastHash {
			continue
		}

		localContent, localErr := e.readNoteFile(note.Filename)
		if localErr == nil {
			localHash := computeHash(localContent)
			if localHash != state.LastHash {
				e.recordPullFailure(noteID, name, fmt.Errorf("local changes conflict with remote changes"))
				continue
			}
		}

		localPath := filepath.Join(e.notesDir, note.Filename)
		if err := atomicWriteLocal(localPath, remoteContent); err != nil {
			e.recordPullFailure(noteID, name, fmt.Errorf("write local file: %w", err))
			continue
		}

		fm, body, fmErr := core.ParseFrontmatter(remoteContent)
		if fmErr != nil {
			e.recordPullFailure(noteID, name, fmt.Errorf("parse frontmatter: %w", fmErr))
			continue
		}
		note.Title = fm.Title
		note.Tags = fm.Tags
		note.Content = remoteContent
		note.ContentHash = computeHash(remoteContent)
		note.WordCount = core.WordCount(body)
		note.ModifiedAt = time.Now().UTC()

		if err := e.store.UpdateNote(note); err != nil {
			e.recordPullFailure(noteID, name, fmt.Errorf("update store: %w", err))
			continue
		}

		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    noteID,
			Backend:   name,
			Direction: "pull",
			Status:    "success",
			SyncedAt:  time.Now().UTC(),
			Hash:      remoteHash,
		})

		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     noteID,
			Backend:    name,
			RemoteID:   state.RemoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   remoteHash,
			Status:     "synced",
		})
	}

	return nil
}

func (e *Engine) PullAll() error {
	notes, err := e.store.ListNotes()
	if err != nil {
		return fmt.Errorf("list notes: %w", err)
	}

	var firstErr error
	for _, note := range notes {
		if err := e.PullNote(note.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func (e *Engine) SyncAll() error {
	notes, err := e.store.ListNotes()
	if err != nil {
		return fmt.Errorf("list notes: %w", err)
	}

	var firstErr error
	for _, note := range notes {
		if err := e.PushNote(note.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (e *Engine) ResolveConflict(noteID, backend, strategy string) error {
	note, err := e.store.GetNote(noteID)
	if err != nil {
		return fmt.Errorf("get note: %w", err)
	}

	conns := e.getConnectors()
	conn, ok := conns[backend]
	if !ok {
		return fmt.Errorf("backend not registered: %s", backend)
	}

	state, err := e.store.GetSyncState(noteID, backend)
	if err != nil {
		return fmt.Errorf("get sync state: %w", err)
	}

	switch strategy {
	case "local":
		content, err := e.readNoteFile(note.Filename)
		if err != nil {
			return fmt.Errorf("read local file: %w", err)
		}
		currentHash := computeHash(content)

		if err := conn.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		remoteID, pushErr := conn.Push(note, content)
		if pushErr != nil {
			return fmt.Errorf("push local to %s: %w", backend, pushErr)
		}

		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    noteID,
			Backend:   backend,
			Direction: "push",
			Status:    "resolved_local",
			SyncedAt:  time.Now().UTC(),
			Hash:      currentHash,
		})
		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     noteID,
			Backend:    backend,
			RemoteID:   remoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   currentHash,
			Status:     "synced",
		})

	case "remote":
		if err := conn.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		remoteContent, pullErr := conn.Pull(state.RemoteID)
		if pullErr != nil {
			return fmt.Errorf("pull from %s: %w", backend, pullErr)
		}
		remoteHash := computeHash(remoteContent)

		localPath := filepath.Join(e.notesDir, note.Filename)
		if err := atomicWriteLocal(localPath, remoteContent); err != nil {
			return fmt.Errorf("write local file: %w", err)
		}

		fm, body, fmErr := core.ParseFrontmatter(remoteContent)
		if fmErr != nil {
			return fmt.Errorf("parse frontmatter: %w", fmErr)
		}
		note.Title = fm.Title
		note.Tags = fm.Tags
		note.Content = remoteContent
		note.ContentHash = computeHash(remoteContent)
		note.WordCount = core.WordCount(body)
		note.ModifiedAt = time.Now().UTC()

		if err := e.store.UpdateNote(note); err != nil {
			return fmt.Errorf("update store: %w", err)
		}

		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    noteID,
			Backend:   backend,
			Direction: "pull",
			Status:    "resolved_remote",
			SyncedAt:  time.Now().UTC(),
			Hash:      remoteHash,
		})
		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     noteID,
			Backend:    backend,
			RemoteID:   state.RemoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   remoteHash,
			Status:     "synced",
		})

	default:
		return fmt.Errorf("unknown conflict resolution strategy: %s", strategy)
	}

	return nil
}

func (e *Engine) SyncStatus(noteID string) ([]*storage.SyncState, error) {
	connectors := e.getConnectors()
	if len(connectors) == 0 {
		return nil, nil
	}

	var states []*storage.SyncState
	for name := range connectors {
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
	connectors := e.getConnectors()
	if len(connectors) == 0 {
		return nil, nil
	}

	states, err := e.store.ListSyncStates()
	if err != nil {
		return nil, err
	}

	connectorNames := make(map[string]bool)
	for name := range connectors {
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

	existing, stateErr := e.store.GetSyncState(noteID, backend)
	if stateErr != nil {
		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:   noteID,
			Backend:  backend,
			Status:   "failed",
			ErrorMsg: err.Error(),
		})
		return
	}

	existing.Status = "failed"
	existing.ErrorMsg = err.Error()
	existing.LastSyncAt = time.Now().UTC()
	e.store.UpsertSyncState(existing)
}

func (e *Engine) recordPullFailure(noteID, backend string, err error) {
	e.store.AddSyncHistory(&storage.SyncHistoryEntry{
		NoteID:    noteID,
		Backend:   backend,
		Direction: "pull",
		Status:    "failed",
		SyncedAt:  time.Now().UTC(),
	})

	existing, stateErr := e.store.GetSyncState(noteID, backend)
	if stateErr != nil {
		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:   noteID,
			Backend:  backend,
			Status:   "failed",
			ErrorMsg: err.Error(),
		})
		return
	}

	existing.Status = "failed"
	existing.ErrorMsg = err.Error()
	existing.LastSyncAt = time.Now().UTC()
	e.store.UpsertSyncState(existing)
}

func (e *Engine) QueueLength() (int, error) {
	return e.store.QueueLength()
}

func (e *Engine) readNoteFile(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("empty filename")
	}
	if strings.Contains(filename, "..") {
		return "", fmt.Errorf("invalid filename: %s", filename)
	}
	path := filepath.Join(e.notesDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func atomicWriteLocal(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vault-sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Sync()
	tmp.Close()

	return os.Rename(tmpName, path)
}

func computeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}
