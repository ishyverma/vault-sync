package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

	GetSyncState(noteID, backend string) (*SyncState, error)
	UpsertSyncState(state *SyncState) error
	DeleteSyncState(noteID, backend string) error
	ListSyncStates() ([]*SyncState, error)
	ListSyncStatesByStatus(status string) ([]*SyncState, error)
	EnqueueSyncJob(noteID string, backends []string, direction string) error
	DequeueSyncJob() (*SyncQueueItem, error)
	QueueLength() (int, error)
	AddSyncHistory(entry *SyncHistoryEntry) error
	ListSyncHistory(noteID string) ([]*SyncHistoryEntry, error)
}

type NoteStore struct {
	db       *sql.DB
	dbPath   string
	vaultDir string
}

func NewNoteStore(vaultDir string) *NoteStore {
	return &NoteStore{
		vaultDir: vaultDir,
		dbPath:   filepath.Join(vaultDir, "vault.db"),
	}
}

func (s *NoteStore) Init() error {
	if err := os.MkdirAll(s.vaultDir, 0o755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", s.dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return err
	}
	s.db = db

	return s.migrate()
}

func (s *NoteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS notes (
		id            TEXT PRIMARY KEY,
		filename      TEXT NOT NULL,
		title         TEXT,
		path          TEXT NOT NULL,
		folder        TEXT DEFAULT '',
		content_hash  TEXT NOT NULL DEFAULT '',
		word_count    INTEGER DEFAULT 0,
		created_at    DATETIME,
		modified_at   DATETIME,
		archived      INTEGER DEFAULT 0,
		pinned        INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS tags (
		note_id  TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		tag      TEXT NOT NULL,
		PRIMARY KEY (note_id, tag)
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts4(
		title,
		content
	);

	CREATE TABLE IF NOT EXISTS sync_state (
		note_id      TEXT NOT NULL,
		backend      TEXT NOT NULL,
		remote_id    TEXT DEFAULT '',
		last_sync_at DATETIME,
		last_hash    TEXT DEFAULT '',
		status       TEXT NOT NULL DEFAULT 'local_only',
		error_msg    TEXT DEFAULT '',
		PRIMARY KEY (note_id, backend)
	);

	CREATE TABLE IF NOT EXISTS sync_queue (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		note_id   TEXT NOT NULL,
		backends  TEXT NOT NULL,
		direction TEXT NOT NULL DEFAULT 'push',
		queued_at DATETIME,
		attempts  INTEGER DEFAULT 0,
		last_error TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS sync_history (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		note_id   TEXT NOT NULL,
		backend   TEXT NOT NULL,
		direction TEXT NOT NULL,
		status    TEXT NOT NULL,
		synced_at DATETIME,
		hash      TEXT DEFAULT ''
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *NoteStore) CreateNote(note *Note) error {
	if note.ID == "" {
		return ErrNoteIDRequired
	}

	now := time.Now().UTC()
	note.CreatedAt = now
	note.ModifiedAt = now

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO notes (id, filename, title, path, folder, content_hash, word_count, created_at, modified_at, archived, pinned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		note.ID, note.Filename, note.Title, note.Path, note.Folder, note.ContentHash, note.WordCount,
		note.CreatedAt.Format(time.RFC3339), note.ModifiedAt.Format(time.RFC3339), boolToInt(note.Archived), boolToInt(note.Pinned))
	if err != nil {
		return err
	}

	if err := s.insertNoteFTS(tx, note); err != nil {
		return err
	}

	for _, tag := range note.Tags {
		_, err = tx.Exec(`INSERT OR IGNORE INTO tags (note_id, tag) VALUES (?, ?)`, note.ID, tag)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *NoteStore) GetNote(id string) (*Note, error) {
	row := s.db.QueryRow(`
		SELECT id, filename, COALESCE(title,''), path, COALESCE(folder,''), COALESCE(content_hash,''),
		       COALESCE(word_count,0), COALESCE(created_at,''), COALESCE(modified_at,''), COALESCE(archived,0), COALESCE(pinned,0)
		FROM notes WHERE id = ?`, id)

	n := &Note{}
	var createdStr, modifiedStr string
	var archivedInt, pinnedInt int
	err := row.Scan(&n.ID, &n.Filename, &n.Title, &n.Path, &n.Folder, &n.ContentHash,
		&n.WordCount, &createdStr, &modifiedStr, &archivedInt, &pinnedInt)
	if err == sql.ErrNoRows {
		return nil, ErrNoteNotFound
	}
	if err != nil {
		return nil, err
	}

	n.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	n.ModifiedAt, _ = time.Parse(time.RFC3339, modifiedStr)
	n.Archived = intToBool(archivedInt)
	n.Pinned = intToBool(pinnedInt)

	tags, err := s.getTags(n.ID)
	if err != nil {
		return nil, err
	}
	n.Tags = tags

	return n, nil
}

func (s *NoteStore) FindNoteByFilename(filename string) (*Note, error) {
	row := s.db.QueryRow(`
		SELECT id FROM notes WHERE filename = ? LIMIT 1`, filename)

	var id string
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return nil, ErrNoteNotFound
	}
	if err != nil {
		return nil, err
	}

	return s.GetNote(id)
}

func (s *NoteStore) UpdateNote(note *Note) error {
	note.ModifiedAt = time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE notes SET filename=?, title=?, path=?, folder=?, content_hash=?, word_count=?,
		                  modified_at=?, archived=?, pinned=?
		WHERE id=?`,
		note.Filename, note.Title, note.Path, note.Folder, note.ContentHash, note.WordCount,
		note.ModifiedAt.Format(time.RFC3339), boolToInt(note.Archived), boolToInt(note.Pinned), note.ID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNoteNotFound
	}

	if err := s.deleteNoteFTS(tx, note.ID); err != nil {
		return err
	}
	if err := s.insertNoteFTS(tx, note); err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM tags WHERE note_id = ?`, note.ID)
	if err != nil {
		return err
	}

	for _, tag := range note.Tags {
		_, err = tx.Exec(`INSERT OR IGNORE INTO tags (note_id, tag) VALUES (?, ?)`, note.ID, tag)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *NoteStore) DeleteNote(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.deleteNoteFTS(tx, id); err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM notes WHERE id = ?`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM tags WHERE note_id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *NoteStore) ListNotes() ([]*Note, error) {
	rows, err := s.db.Query(`
		SELECT id, filename, COALESCE(title,''), path, COALESCE(folder,''), COALESCE(content_hash,''),
		       COALESCE(word_count,0), created_at, modified_at, COALESCE(archived,0), COALESCE(pinned,0)
		FROM notes WHERE archived = 0
		ORDER BY modified_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanNotes(rows)
}

func (s *NoteStore) ListNotesByTag(tag string) ([]*Note, error) {
	rows, err := s.db.Query(`
		SELECT n.id, n.filename, COALESCE(n.title,''), n.path, COALESCE(n.folder,''), COALESCE(n.content_hash,''),
		       COALESCE(n.word_count,0), n.created_at, n.modified_at, COALESCE(n.archived,0), COALESCE(n.pinned,0)
		FROM notes n
		INNER JOIN tags t ON t.note_id = n.id
		WHERE n.archived = 0 AND t.tag = ?
		ORDER BY n.modified_at DESC`, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanNotes(rows)
}

func (s *NoteStore) SearchNotes(query string) ([]*Note, error) {
	if query == "" {
		return s.ListNotes()
	}

	rows, err := s.db.Query(`
		SELECT n.id, n.filename, COALESCE(n.title,''), n.path, COALESCE(n.folder,''), COALESCE(n.content_hash,''),
		       COALESCE(n.word_count,0), n.created_at, n.modified_at, COALESCE(n.archived,0), COALESCE(n.pinned,0)
		FROM notes_fts f
		INNER JOIN notes n ON n.rowid = f.docid
		WHERE notes_fts MATCH ? AND n.archived = 0
		ORDER BY n.modified_at DESC
		LIMIT 50`, query)
	if err != nil {
		if rows != nil {
			rows.Close()
		}
		rows, err = s.db.Query(`
			SELECT id, filename, COALESCE(title,''), path, COALESCE(folder,''), COALESCE(content_hash,''),
			       COALESCE(word_count,0), created_at, modified_at, COALESCE(archived,0), COALESCE(pinned,0)
			FROM notes
			WHERE archived = 0 AND (title LIKE '%' || ? || '%' OR filename LIKE '%' || ? || '%')
			ORDER BY modified_at DESC
			LIMIT 50`, query, query)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	return s.scanNotes(rows)
}

func (s *NoteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *NoteStore) getTags(noteID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT tag FROM tags WHERE note_id = ? ORDER BY tag`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (s *NoteStore) scanNotes(rows *sql.Rows) ([]*Note, error) {
	var notes []*Note
	for rows.Next() {
		n := &Note{}
		var createdStr, modifiedStr string
		var archivedInt, pinnedInt int
		err := rows.Scan(&n.ID, &n.Filename, &n.Title, &n.Path, &n.Folder, &n.ContentHash,
			&n.WordCount, &createdStr, &modifiedStr, &archivedInt, &pinnedInt)
		if err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		n.ModifiedAt, _ = time.Parse(time.RFC3339, modifiedStr)
		n.Archived = intToBool(archivedInt)
		n.Pinned = intToBool(pinnedInt)

		tags, err := s.getTags(n.ID)
		if err != nil {
			return nil, err
		}
		n.Tags = tags
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

func (s *NoteStore) insertNoteFTS(tx *sql.Tx, note *Note) error {
	var rowid int64
	err := tx.QueryRow(`SELECT rowid FROM notes WHERE id = ?`, note.ID).Scan(&rowid)
	if err != nil {
		return err
	}
	bodyText := note.Content
	if bodyText == "" {
		bodyText = note.Title
	}
	_, err = tx.Exec(`INSERT INTO notes_fts(docid, title, content) VALUES (?, ?, ?)`,
		rowid, note.Title, bodyText)
	return err
}

func (s *NoteStore) deleteNoteFTS(tx *sql.Tx, id string) error {
	var rowid int64
	err := tx.QueryRow(`SELECT rowid FROM notes WHERE id = ?`, id).Scan(&rowid)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM notes_fts WHERE docid = ?`, rowid)
	return err
}
