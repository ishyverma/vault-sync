package storage

import "errors"

var (
	ErrNoteNotFound      = errors.New("note not found")
	ErrNoteIDRequired    = errors.New("note ID is required")
	ErrSyncStateNotFound = errors.New("sync state not found")
	ErrSyncJobNotFound   = errors.New("sync job not found")
)
