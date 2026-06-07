package storage

import (
	"database/sql"
	"time"
)

type Version struct {
	ID         int64     `json:"id"`
	NoteID     string    `json:"note_id"`
	VersionNum int       `json:"version_num"`
	Content    string    `json:"content"`
	SavedAt    time.Time `json:"saved_at"`
	Trigger    string    `json:"trigger"`
}

func (s *NoteStore) SaveVersion(noteID string, content string, trigger string) (*Version, error) {
	var maxVer int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version_num),0) FROM versions WHERE note_id = ?`, noteID).Scan(&maxVer)
	if err != nil {
		return nil, err
	}

	v := &Version{
		NoteID:     noteID,
		VersionNum: maxVer + 1,
		Content:    content,
		SavedAt:    time.Now().UTC(),
		Trigger:    trigger,
	}

	result, err := s.db.Exec(`
		INSERT INTO versions (note_id, version_num, content, saved_at, trigger)
		VALUES (?, ?, ?, ?, ?)`,
		v.NoteID, v.VersionNum, v.Content, v.SavedAt.Format(time.RFC3339), v.Trigger)
	if err != nil {
		return nil, err
	}

	v.ID, _ = result.LastInsertId()
	return v, nil
}

func (s *NoteStore) ListVersions(noteID string) ([]*Version, error) {
	rows, err := s.db.Query(`
		SELECT id, note_id, version_num, content, saved_at, COALESCE(trigger,'save')
		FROM versions WHERE note_id = ? ORDER BY version_num DESC`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*Version
	for rows.Next() {
		v := &Version{}
		var savedAt string
		if err := rows.Scan(&v.ID, &v.NoteID, &v.VersionNum, &v.Content, &savedAt, &v.Trigger); err != nil {
			return nil, err
		}
		v.SavedAt, _ = time.Parse(time.RFC3339, savedAt)
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (s *NoteStore) GetVersion(noteID string, versionNum int) (*Version, error) {
	v := &Version{}
	var savedAt string
	err := s.db.QueryRow(`
		SELECT id, note_id, version_num, content, saved_at, COALESCE(trigger,'save')
		FROM versions WHERE note_id = ? AND version_num = ?`, noteID, versionNum).
		Scan(&v.ID, &v.NoteID, &v.VersionNum, &v.Content, &savedAt, &v.Trigger)
	if err == sql.ErrNoRows {
		return nil, ErrNoteNotFound
	}
	if err != nil {
		return nil, err
	}
	v.SavedAt, _ = time.Parse(time.RFC3339, savedAt)
	return v, nil
}
