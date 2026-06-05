package storage

import "time"

type Note struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Title       string    `json:"title"`
	Path        string    `json:"path"`
	Folder      string    `json:"folder"`
	ContentHash string    `json:"content_hash"`
	WordCount   int       `json:"word_count"`
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
	Archived    bool      `json:"archived"`
	Pinned      bool      `json:"pinned"`
	Content     string    `json:"-"` // full text for FTS indexing (not stored in notes table)
	Tags        []string  `json:"tags"`
}
