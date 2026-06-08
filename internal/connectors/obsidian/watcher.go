package obsidian

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors an Obsidian vault directory for file changes.
type Watcher struct {
	vaultPath string
	subfolder string
	onChange  func(filename string)
	watcher   *fsnotify.Watcher
	done      chan struct{}
}

// NewWatcher creates a new Obsidian vault watcher.
func NewWatcher(vaultPath, subfolder string, onChange func(filename string)) *Watcher {
	return &Watcher{
		vaultPath: vaultPath,
		subfolder: subfolder,
		onChange:  onChange,
		done:      make(chan struct{}),
	}
}

// Start begins watching the vault directory for changes.
func (w *Watcher) Start() error {
	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	targetDir := filepath.Join(w.vaultPath, w.subfolder)
	if err := w.watcher.Add(targetDir); err != nil {
		return fmt.Errorf("watch %s: %w", targetDir, err)
	}

	go w.loop()
	return nil
}

// Stop stops the file watcher.
func (w *Watcher) Stop() error {
	close(w.done)
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

func (w *Watcher) loop() {
	var timer *time.Timer
	var pending string

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			if timer != nil {
				timer.Stop()
			}
			pending = event.Name
			timer = time.AfterFunc(500*time.Millisecond, func() {
				if w.onChange != nil && pending != "" {
					relPath, err := filepath.Rel(filepath.Join(w.vaultPath, w.subfolder), pending)
					if err == nil {
						w.onChange(relPath)
					}
				}
				pending = ""
			})

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("obsidian watcher error: %v", err)

		case <-w.done:
			return
		}
	}
}
