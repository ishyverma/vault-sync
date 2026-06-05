package storage

import "errors"

var (
	ErrNoteNotFound   = errors.New("note not found")
	ErrNoteIDRequired = errors.New("note ID is required")
)
