package storage

import "errors"

var (
	ErrNoteNotFound      = errors.New("note not found")
	ErrNoteAlreadyExists = errors.New("note already exists")
)
