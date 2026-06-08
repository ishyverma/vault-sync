package sync

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ishyverma/vault-sync/internal/connectors"
	"github.com/ishyverma/vault-sync/internal/connectors/notion"
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
	mu               sync.RWMutex
	store            *storage.NoteStore
	connectors       map[string]connectors.Connector
	notesDir         string
	retryLimit       int
	conflictStrategy string
	pushStrategy     string
	pullStrategy     string
	preSyncHook      string
	postSyncHook     string
	onConflictHook   string
}

func NewEngine(store *storage.NoteStore, notesDir string) *Engine {
	return &Engine{
		store:        store,
		connectors:   make(map[string]connectors.Connector),
		notesDir:     notesDir,
		retryLimit:   5,
		pushStrategy: "local_wins",
		pullStrategy: "remote_wins",
	}
}

func (e *Engine) SetRetryLimit(limit int) {
	e.retryLimit = limit
}

func (e *Engine) SetConflictStrategy(strategy string) {
	e.conflictStrategy = strategy
}

func (e *Engine) SetPushStrategy(strategy string) {
	e.pushStrategy = strategy
}

func (e *Engine) SetPullStrategy(strategy string) {
	e.pullStrategy = strategy
}

func (e *Engine) SetHooks(preSync, postSync, onConflict string) {
	e.preSyncHook = preSync
	e.postSyncHook = postSync
	e.onConflictHook = onConflict
}

func (e *Engine) ExecutePreSyncHook() error {
	return e.executeHook(e.preSyncHook)
}

func (e *Engine) ExecutePostSyncHook() error {
	return e.executeHook(e.postSyncHook)
}

func (e *Engine) executeHook(cmd string) error {
	if cmd == "" {
		return nil
	}
	c := exec.Command("sh", "-c", cmd)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook %q failed (%v): %s", cmd, err, strings.TrimSpace(string(out)))
	}
	if len(out) > 0 {
		log.Printf("hook %q: %s", cmd, strings.TrimSpace(string(out)))
	}
	return nil
}

func (e *Engine) RegisterConnector(name string, c connectors.Connector) {
	if c == nil {
		panic("sync: RegisterConnector called with nil connector")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connectors[name] = c
}

func (e *Engine) Connectors() map[string]connectors.Connector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	cp := make(map[string]connectors.Connector, len(e.connectors))
	for k, v := range e.connectors {
		cp[k] = v
	}
	return cp
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

	rawHash := computeHash(content)

	// Only save version if content actually changed
	if rawHash != note.ContentHash {
		if _, err := e.store.SaveVersion(noteID, content, "pre_sync"); err != nil {
			return fmt.Errorf("save version: %w", err)
		}
	}

	// Parse frontmatter
	fm, body, fmErr := core.ParseFrontmatter(content)
	if fmErr != nil {
		body = content
	} else if body == "" {
		body = content
	}

	// Update note metadata if content changed
	if rawHash != note.ContentHash {
		note.Content = content
		note.ContentHash = rawHash
		note.WordCount = core.WordCount(body)
		if fmErr == nil && fm.Title != "" {
			note.Title = fm.Title
			note.Tags = fm.Tags
		}
		if err := e.store.UpdateNote(note); err != nil {
			return fmt.Errorf("update note metadata: %w", err)
		}
	}

	var backendErrs []string

	for name, conn := range e.getConnectors() {
		canonicalHash := e.canonicalHash(name, fm, body, rawHash)

		state, stateErr := e.store.GetSyncState(noteID, name)

		// If synced and hash matches, skip
		if stateErr == nil && state.LastHash == canonicalHash && state.Status == "synced" {
			continue
		}

		if err := conn.Connect(); err != nil {
			e.recordFailure(noteID, name, err)
			backendErrs = append(backendErrs, fmt.Sprintf("[%s] connect: %v", name, err))
			continue
		}

		// Check for remote changes (conflict detection)
		if stateErr == nil && state.RemoteID != "" {
			remoteChanged, checkErr := e.detectConflict(conn, state)
			if checkErr != nil {
				e.recordFailure(noteID, name, fmt.Errorf("conflict check: %w", checkErr))
				backendErrs = append(backendErrs, fmt.Sprintf("[%s] conflict check: %v", name, checkErr))
				continue
			}
			if remoteChanged {
				// Conflict detected
				e.store.UpsertSyncState(&storage.SyncState{
					NoteID:   noteID,
					Backend:  name,
					RemoteID: state.RemoteID,
					LastHash: state.LastHash,
					Status:   "conflict",
					ErrorMsg: "remote file modified externally",
				})
				if err := e.executeHook(e.onConflictHook); err != nil {
					log.Printf("onConflict hook: %v", err)
				}

				// Apply push conflict strategy (default: local_wins)
				if err := e.applyConflictStrategy(note, conn, name, content, canonicalHash, state.RemoteID, state.LastSyncAt, e.pushStrategy); err != nil {
					backendErrs = append(backendErrs, fmt.Sprintf("[%s] conflict strategy: %v", name, err))
				}
				continue
			}
		}

		existingID := ""
		if stateErr == nil {
			existingID = state.RemoteID
		}

		remoteID, pushErr := conn.Push(note, content, existingID)
		if pushErr != nil {
			if isConnectivityError(pushErr) {
				e.store.EnqueueSyncJob(noteID, []string{name}, "push", 0)
			} else {
				e.recordFailure(noteID, name, pushErr)
			}
			backendErrs = append(backendErrs, fmt.Sprintf("[%s] %v", name, pushErr))
			continue
		}

		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    noteID,
			Backend:   name,
			Direction: "push",
			Status:    "success",
			SyncedAt:  time.Now().UTC(),
			Hash:      rawHash,
		})

		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     noteID,
			Backend:    name,
			RemoteID:   remoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   canonicalHash,
			Status:     "synced",
		})
	}

	if len(backendErrs) > 0 {
		return fmt.Errorf("push errors: %s", strings.Join(backendErrs, "; "))
	}

	return nil
}

func (e *Engine) applyConflictStrategy(note *storage.Note, conn connectors.Connector, backend, content, canonicalHash, remoteID string, lastSyncAt time.Time, strategy string) error {
	if strategy == "" {
		strategy = e.conflictStrategy
	}
	switch strategy {
	case "local_wins":
		_, pushErr := conn.Push(note, content, remoteID)
		if pushErr != nil {
			return fmt.Errorf("local_wins push: %w", pushErr)
		}
		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    note.ID,
			Backend:   backend,
			Direction: "push",
			Status:    "resolved_local",
			SyncedAt:  time.Now().UTC(),
		})
		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     note.ID,
			Backend:    backend,
			RemoteID:   remoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   canonicalHash,
			Status:     "synced",
		})
		return nil

	case "remote_wins":
		remoteContent, pullErr := conn.Pull(remoteID)
		if pullErr != nil {
			return fmt.Errorf("remote_wins pull: %w", pullErr)
		}
		localPath := filepath.Join(e.notesDir, note.Filename)
		if err := atomicWriteLocal(localPath, remoteContent); err != nil {
			return fmt.Errorf("remote_wins write: %w", err)
		}
		remoteFm, remoteBody, _ := core.ParseFrontmatter(remoteContent)
		remoteHash := computeHash(remoteContent)
		note.Title = remoteFm.Title
		note.Tags = remoteFm.Tags
		note.Content = remoteContent
		note.ContentHash = remoteHash
		note.WordCount = core.WordCount(remoteBody)
		note.ModifiedAt = time.Now().UTC()
		if err := e.store.UpdateNote(note); err != nil {
			return fmt.Errorf("remote_wins update store: %w", err)
		}
		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    note.ID,
			Backend:   backend,
			Direction: "pull",
			Status:    "resolved_remote",
			SyncedAt:  time.Now().UTC(),
		})
		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     note.ID,
			Backend:    backend,
			RemoteID:   remoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   remoteHash,
			Status:     "synced",
		})
		return nil

	case "last_write_wins":
		localPath := filepath.Join(e.notesDir, note.Filename)
		localInfo, statErr := os.Stat(localPath)
		if statErr == nil && !lastSyncAt.IsZero() && localInfo.ModTime().Before(lastSyncAt) {
			// Remote was modified more recently — pull remote
			remoteContent, pullErr := conn.Pull(remoteID)
			if pullErr != nil {
				return fmt.Errorf("last_write_wins pull: %w", pullErr)
			}
			if err := atomicWriteLocal(localPath, remoteContent); err != nil {
				return fmt.Errorf("last_write_wins write: %w", err)
			}
			remoteFm, remoteBody, _ := core.ParseFrontmatter(remoteContent)
			remoteHash := computeHash(remoteContent)
			note.Title = remoteFm.Title
			note.Tags = remoteFm.Tags
			note.Content = remoteContent
			note.ContentHash = remoteHash
			note.WordCount = core.WordCount(remoteBody)
			note.ModifiedAt = time.Now().UTC()
			if err := e.store.UpdateNote(note); err != nil {
				return fmt.Errorf("last_write_wins update store: %w", err)
			}
			e.store.AddSyncHistory(&storage.SyncHistoryEntry{
				NoteID:    note.ID,
				Backend:   backend,
				Direction: "pull",
				Status:    "resolved_remote",
				SyncedAt:  time.Now().UTC(),
			})
			e.store.UpsertSyncState(&storage.SyncState{
				NoteID:     note.ID,
				Backend:    backend,
				RemoteID:   remoteID,
				LastSyncAt: time.Now().UTC(),
				LastHash:   remoteHash,
				Status:     "synced",
			})
			return nil
		}
		// Local was modified more recently — push local
		_, pushErr := conn.Push(note, content, remoteID)
		if pushErr != nil {
			return fmt.Errorf("last_write_wins push: %w", pushErr)
		}
		e.store.AddSyncHistory(&storage.SyncHistoryEntry{
			NoteID:    note.ID,
			Backend:   backend,
			Direction: "push",
			Status:    "resolved_local",
			SyncedAt:  time.Now().UTC(),
		})
		e.store.UpsertSyncState(&storage.SyncState{
			NoteID:     note.ID,
			Backend:    backend,
			RemoteID:   remoteID,
			LastSyncAt: time.Now().UTC(),
			LastHash:   canonicalHash,
			Status:     "synced",
		})
		return nil

	default:
		// "ask" — leave as conflict for user to resolve
		return nil
	}
}

func (e *Engine) canonicalHash(backend string, fm core.Frontmatter, body string, rawHash string) string {
	if backend == "notion" {
		processedBody := notion.EmbedTags(body, fm.Tags)
		return computeHash(core.BuildNoteContent(fm, processedBody))
	}
	return rawHash
}

func (e *Engine) ProcessQueue() (int, error) {
	var processed int
	const maxProcessed = 1000
	for processed < maxProcessed {
		item, err := e.store.DequeueSyncJob()
		if err != nil {
			if errors.Is(err, storage.ErrSyncJobNotFound) {
				break
			}
			return processed, err
		}

		if item.Attempts >= e.retryLimit {
			e.recordFailure(item.NoteID, strings.Join(item.Backends, ","), fmt.Errorf("retry limit exceeded (%d)", e.retryLimit))
			processed++
			continue
		}

		var processErr error
		if item.Direction == "push" {
			processErr = e.PushNote(item.NoteID)
		} else {
			processErr = e.PullNote(item.NoteID)
		}

		if processErr != nil {
			if isConnectivityError(processErr) {
				e.store.EnqueueSyncJob(item.NoteID, item.Backends, item.Direction, item.Attempts+1)
			} else {
				e.recordFailure(item.NoteID, strings.Join(item.Backends, ","), processErr)
			}
		}
		processed++
	}
	return processed, nil
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

		if err := conn.Connect(); err != nil {
			e.recordPullFailure(noteID, name, err)
			continue
		}

		remoteContent, pullErr := conn.Pull(state.RemoteID)
		if pullErr != nil {
			if errors.Is(pullErr, notion.ErrNotFound) || os.IsNotExist(pullErr) {
				continue
			}
			e.recordPullFailure(noteID, name, pullErr)
			continue
		}

		remoteFm, remoteBody, _ := core.ParseFrontmatter(remoteContent)
		remoteHash := computeHash(remoteContent)
		remoteCanonical := e.canonicalHash(name, remoteFm, remoteBody, remoteHash)
		if remoteCanonical == state.LastHash {
			continue
		}

		localContent, localErr := e.readNoteFile(note.Filename)
		if localErr == nil {
			localFm, localBody, _ := core.ParseFrontmatter(localContent)
			localCanonical := e.canonicalHash(name, localFm, localBody, computeHash(localContent))
			if localCanonical != state.LastHash {
				e.store.UpsertSyncState(&storage.SyncState{
					NoteID:   noteID,
					Backend:  name,
					RemoteID: state.RemoteID,
					LastHash: state.LastHash,
					Status:   "conflict",
					ErrorMsg: "local changes conflict with remote changes",
				})
				if err := e.executeHook(e.onConflictHook); err != nil {
					log.Printf("onConflict hook: %v", err)
				}

				// Save local version then auto-resolve with pull strategy (default: remote_wins)
				if localContent != "" {
					e.store.SaveVersion(noteID, localContent, "pre_pull")
				}
				localHash := computeHash(localContent)
				if err := e.applyConflictStrategy(note, conn, name, localContent, localHash, state.RemoteID, state.LastSyncAt, e.pullStrategy); err != nil {
					log.Printf("pull conflict auto-resolve: %v", err)
				}
				continue
			}
		}

		if localContent != "" {
			if _, err := e.store.SaveVersion(noteID, localContent, "pre_pull"); err != nil {
				e.recordPullFailure(noteID, name, fmt.Errorf("save pre-pull version: %w", err))
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
		if body == "" {
			body = remoteContent
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
			LastHash:   remoteCanonical,
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
	if err := e.executeHook(e.preSyncHook); err != nil {
		log.Printf("preSync hook: %v", err)
	}
	defer func() {
		if err := e.executeHook(e.postSyncHook); err != nil {
			log.Printf("postSync hook: %v", err)
		}
	}()

	notes, err := e.store.ListNotes()
	if err != nil {
		return fmt.Errorf("list notes: %w", err)
	}

	var firstErr error

	e.ProcessQueue()

	for _, note := range notes {
		if err := e.PushNote(note.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	e.ProcessQueue()

	for _, note := range notes {
		if err := e.PullNote(note.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	e.ProcessQueue()

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

	content, _ := e.readNoteFile(note.Filename)
	if content != "" {
		e.store.SaveVersion(noteID, content, "pre_resolve:"+strategy)
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
		remoteID, pushErr := conn.Push(note, content, state.RemoteID)
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
		if body == "" {
			body = remoteContent
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
		if errors.Is(err, notion.ErrNotFound) {
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

func (e *Engine) ListConflicts() ([]*storage.SyncState, error) {
	return e.store.ListSyncStatesByStatus("conflict")
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
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}

func isConnectivityError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "tls handshake timeout")
}

func computeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}
