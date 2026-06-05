package connectors

import "github.com/ishyverma/vault-sync/internal/storage"

type Connector interface {
	Connect() error
	Push(note *storage.Note, content string) (remoteID string, err error)
	Pull(remoteID string) (content string, err error)
	Delete(remoteID string) error
	Status() (healthy bool, err error)
	Name() string
}
