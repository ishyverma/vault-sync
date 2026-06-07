package storage

import (
	"database/sql"
	"encoding/json"
	"time"
)

type SyncState struct {
	NoteID     string    `json:"note_id"`
	Backend    string    `json:"backend"`
	RemoteID   string    `json:"remote_id,omitempty"`
	LastSyncAt time.Time `json:"last_sync_at,omitempty"`
	LastHash   string    `json:"last_hash,omitempty"`
	Status     string    `json:"status"`
	ErrorMsg   string    `json:"error_msg,omitempty"`
}

type SyncQueueItem struct {
	ID        int64     `json:"id"`
	NoteID    string    `json:"note_id"`
	Backends  []string  `json:"backends"`
	Direction string    `json:"direction"`
	QueuedAt  time.Time `json:"queued_at"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
}

type SyncHistoryEntry struct {
	ID        int64     `json:"id"`
	NoteID    string    `json:"note_id"`
	Backend   string    `json:"backend"`
	Direction string    `json:"direction"`
	Status    string    `json:"status"`
	SyncedAt  time.Time `json:"synced_at"`
	Hash      string    `json:"hash,omitempty"`
}

func (s *NoteStore) GetSyncState(noteID, backend string) (*SyncState, error) {
	row := s.db.QueryRow(`
		SELECT note_id, backend, COALESCE(remote_id,''), COALESCE(last_sync_at,''),
		       COALESCE(last_hash,''), status, COALESCE(error_msg,'')
		FROM sync_state WHERE note_id = ? AND backend = ?`, noteID, backend)

	state := &SyncState{}
	var syncAt string
	err := row.Scan(&state.NoteID, &state.Backend, &state.RemoteID, &syncAt,
		&state.LastHash, &state.Status, &state.ErrorMsg)
	if err == sql.ErrNoRows {
		return nil, ErrSyncStateNotFound
	}
	if err != nil {
		return nil, err
	}
	state.LastSyncAt, _ = time.Parse(time.RFC3339, syncAt)
	return state, nil
}

func (s *NoteStore) UpsertSyncState(state *SyncState) error {
	_, err := s.db.Exec(`
		INSERT INTO sync_state (note_id, backend, remote_id, last_sync_at, last_hash, status, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(note_id, backend) DO UPDATE SET
			remote_id=excluded.remote_id, last_sync_at=excluded.last_sync_at,
			last_hash=excluded.last_hash, status=excluded.status, error_msg=excluded.error_msg`,
		state.NoteID, state.Backend, state.RemoteID,
		state.LastSyncAt.Format(time.RFC3339), state.LastHash,
		state.Status, state.ErrorMsg)
	return err
}

func (s *NoteStore) DeleteSyncState(noteID, backend string) error {
	res, err := s.db.Exec(`DELETE FROM sync_state WHERE note_id = ? AND backend = ?`, noteID, backend)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrSyncStateNotFound
	}
	return nil
}

func (s *NoteStore) ListSyncStates() ([]*SyncState, error) {
	rows, err := s.db.Query(`
		SELECT note_id, backend, COALESCE(remote_id,''), COALESCE(last_sync_at,''),
		       COALESCE(last_hash,''), status, COALESCE(error_msg,'')
		FROM sync_state ORDER BY backend, note_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSyncStates(rows)
}

func (s *NoteStore) ListSyncStatesByStatus(status string) ([]*SyncState, error) {
	rows, err := s.db.Query(`
		SELECT note_id, backend, COALESCE(remote_id,''), COALESCE(last_sync_at,''),
		       COALESCE(last_hash,''), status, COALESCE(error_msg,'')
		FROM sync_state WHERE status = ? ORDER BY backend, note_id`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSyncStates(rows)
}

func backoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	d := time.Duration(1<<min(attempt-1, 5)) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func (s *NoteStore) EnqueueSyncJob(noteID string, backends []string, direction string, attempts int) error {
	data, _ := json.Marshal(backends)
	queuedAt := time.Now().UTC()
	if attempts > 0 {
		queuedAt = queuedAt.Add(backoffDelay(attempts))
	}
	_, err := s.db.Exec(`
		INSERT INTO sync_queue (note_id, backends, direction, queued_at, attempts)
		VALUES (?, ?, ?, ?, ?)`, noteID, string(data), direction, queuedAt.Format(time.RFC3339), attempts)
	return err
}

func (s *NoteStore) DequeueSyncJob() (*SyncQueueItem, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	item := &SyncQueueItem{}
	var backendsJSON, queuedAt string
	now := time.Now().UTC().Format(time.RFC3339)
	err = tx.QueryRow(`
		SELECT id, note_id, backends, direction, queued_at, attempts, COALESCE(last_error,'')
		FROM sync_queue WHERE queued_at <= ? ORDER BY id ASC LIMIT 1`, now).Scan(
		&item.ID, &item.NoteID, &backendsJSON, &item.Direction, &queuedAt, &item.Attempts, &item.LastError)
	if err == sql.ErrNoRows {
		return nil, ErrSyncJobNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(backendsJSON), &item.Backends)
	item.QueuedAt, _ = time.Parse(time.RFC3339, queuedAt)

	_, err = tx.Exec(`DELETE FROM sync_queue WHERE id = ?`, item.ID)
	if err != nil {
		return nil, err
	}

	return item, tx.Commit()
}

func (s *NoteStore) QueueLength() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sync_queue`).Scan(&count)
	return count, err
}

func (s *NoteStore) AddSyncHistory(entry *SyncHistoryEntry) error {
	_, err := s.db.Exec(`
		INSERT INTO sync_history (note_id, backend, direction, status, synced_at, hash)
		VALUES (?, ?, ?, ?, ?, ?)`,
		entry.NoteID, entry.Backend, entry.Direction, entry.Status,
		entry.SyncedAt.Format(time.RFC3339), entry.Hash)
	return err
}

func (s *NoteStore) ListRecentSyncHistory(limit int) ([]*SyncHistoryEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, note_id, backend, direction, status, synced_at, COALESCE(hash,'')
		FROM sync_history ORDER BY synced_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*SyncHistoryEntry
	for rows.Next() {
		e := &SyncHistoryEntry{}
		var syncAt string
		if err := rows.Scan(&e.ID, &e.NoteID, &e.Backend, &e.Direction, &e.Status, &syncAt, &e.Hash); err != nil {
			return nil, err
		}
		e.SyncedAt, _ = time.Parse(time.RFC3339, syncAt)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *NoteStore) ListSyncHistory(noteID string) ([]*SyncHistoryEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, note_id, backend, direction, status, synced_at, COALESCE(hash,'')
		FROM sync_history WHERE note_id = ? ORDER BY synced_at DESC LIMIT 50`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*SyncHistoryEntry
	for rows.Next() {
		e := &SyncHistoryEntry{}
		var syncAt string
		if err := rows.Scan(&e.ID, &e.NoteID, &e.Backend, &e.Direction, &e.Status, &syncAt, &e.Hash); err != nil {
			return nil, err
		}
		e.SyncedAt, _ = time.Parse(time.RFC3339, syncAt)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func scanSyncStates(rows *sql.Rows) ([]*SyncState, error) {
	var states []*SyncState
	for rows.Next() {
		s := &SyncState{}
		var syncAt string
		if err := rows.Scan(&s.NoteID, &s.Backend, &s.RemoteID, &syncAt, &s.LastHash, &s.Status, &s.ErrorMsg); err != nil {
			return nil, err
		}
		s.LastSyncAt, _ = time.Parse(time.RFC3339, syncAt)
		states = append(states, s)
	}
	return states, rows.Err()
}
