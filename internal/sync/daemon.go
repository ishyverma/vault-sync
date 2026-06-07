package sync

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
)

type Daemon struct {
	engine      *Engine
	notesDir    string
	obsidianDir string
	pidPath     string
	interval    time.Duration
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	syncCh       chan struct{}
}

func NewDaemon(engine *Engine, notesDir string, interval time.Duration) *Daemon {
	return &Daemon{
		engine:   engine,
		notesDir: notesDir,
		pidPath:  filepath.Join(filepath.Dir(notesDir), "vaultd.pid"),
		interval: interval,
		syncCh:   make(chan struct{}, 1),
	}
}

func (d *Daemon) SetObsidianDir(dir string) {
	d.obsidianDir = dir
}

func (d *Daemon) Start() error {
	if err := d.writePID(); err != nil {
		return fmt.Errorf("write PID: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(d.notesDir); err != nil {
		return fmt.Errorf("watch notes dir: %w", err)
	}

	hasObsidian := false
	if d.obsidianDir != "" {
		if info, err := os.Stat(d.obsidianDir); err == nil && info.IsDir() {
			if err := watcher.Add(d.obsidianDir); err != nil {
				return fmt.Errorf("watch obsidian dir: %w", err)
			}
			hasObsidian = true
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("vaultd started — watching %s (poll: %v, obsidian: %v)", d.notesDir, d.interval, hasObsidian)

	d.wg.Add(3)
	go d.watchLoop(ctx, watcher)
	go d.pollLoop(ctx)
	go d.syncRunner(ctx)

	select {
	case sig := <-sigCh:
		log.Printf("vaultd received signal: %v — shutting down", sig)
	case <-ctx.Done():
	}

	cancel()
	d.wg.Wait()
	d.removePID()

	log.Println("vaultd stopped")
	return nil
}

func (d *Daemon) Stop() error {
	pid, err := d.readPID()
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		d.removePID()
		return nil
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		d.removePID()
		return fmt.Errorf("signal daemon: %w", err)
	}

	return nil
}

func (d *Daemon) Status() (bool, int, error) {
	pid, err := d.readPID()
	if err != nil {
		return false, 0, nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		d.removePID()
		return false, 0, nil
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		d.removePID()
		return false, 0, nil
	}

	return true, pid, nil
}

func (d *Daemon) watchLoop(ctx context.Context, watcher *fsnotify.Watcher) {
	defer d.wg.Done()

	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time
	var pendingEvents []fsnotify.Event

	const debounce = 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !isNoteEvent(event) {
				continue
			}

			pendingEvents = append(pendingEvents, event)
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(debounce)
			debounceCh = debounceTimer.C

		case <-debounceCh:
			d.processEvents(pendingEvents)
			pendingEvents = nil
			debounceTimer = nil
			debounceCh = nil

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)

		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

func (d *Daemon) processEvents(events []fsnotify.Event) {
	var pushNotes, pullNotes []string

	for _, ev := range events {
		filename := filepath.Base(ev.Name)
		dir := filepath.Dir(ev.Name)

		if dir == d.notesDir {
			pushNotes = append(pushNotes, filename)
		} else if d.obsidianDir != "" && (dir == d.obsidianDir || strings.HasPrefix(dir, d.obsidianDir+string(filepath.Separator))) {
			pullNotes = append(pullNotes, filename)
		}
	}

	pushNotes = unique(pushNotes)
	pullNotes = unique(pullNotes)

	common := intersect(pushNotes, pullNotes)
	if len(common) > 0 {
		for _, fn := range common {
			note, err := d.engine.store.FindNoteByFilename(fn)
			if err != nil {
				continue
			}
			localHash, remoteHash := d.compareNoteHashes(note)
			if localHash == remoteHash {
				continue
			}
			if err := d.engine.PullNote(note.ID); err != nil {
				log.Printf("sync %s (common): %v", fn, err)
			}
		}
	}

	pushOnly := subtract(pushNotes, pullNotes)
	pullOnly := subtract(pullNotes, pushNotes)

	for _, fn := range pushOnly {
		note, err := d.engine.store.FindNoteByFilename(fn)
		if err != nil {
			log.Printf("unknown file %s — skipping push", fn)
			continue
		}
		if err := d.engine.PushNote(note.ID); err != nil {
			log.Printf("push %s: %v", fn, err)
		}
	}

	for _, fn := range pullOnly {
		note, err := d.engine.store.FindNoteByFilename(fn)
		if err != nil {
			log.Printf("new obsidian file %s — importing", fn)
			d.importFromObsidian(fn)
			continue
		}
		if err := d.engine.PullNote(note.ID); err != nil {
			log.Printf("pull %s: %v", fn, err)
		}
	}
}

func (d *Daemon) compareNoteHashes(note *storage.Note) (string, string) {
	state, err := d.engine.store.GetSyncState(note.ID, "obsidian")
	if err != nil {
		return "", ""
	}
	return note.ContentHash, state.LastHash
}

func (d *Daemon) importFromObsidian(filename string) {
	remotePath := filepath.Join(d.obsidianDir, filename)
	data, err := os.ReadFile(remotePath)
	if err != nil {
		log.Printf("import %s: read error: %v", filename, err)
		return
	}

	content := string(data)
	localPath := filepath.Join(d.notesDir, filename)
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		log.Printf("import %s: write error: %v", filename, err)
		return
	}

	fm, body, parseErr := core.ParseFrontmatter(content)
	if parseErr != nil {
		log.Printf("import %s: parse error: %v", filename, parseErr)
		return
	}

	h := sha256.Sum256([]byte(content + filename))
	noteID := fmt.Sprintf("%x", h)

	note := &storage.Note{
		ID:          noteID,
		Filename:    filename,
		Title:       fm.Title,
		Path:        filename,
		Content:     content,
		ContentHash: computeHash(content),
		WordCount:   core.WordCount(body),
		CreatedAt:   time.Now().UTC(),
		ModifiedAt:  time.Now().UTC(),
		Tags:        fm.Tags,
	}

	if err := d.engine.store.CreateNote(note); err != nil {
		log.Printf("import %s: db error: %v", filename, err)
		return
	}

	d.engine.store.UpsertSyncState(&storage.SyncState{
		NoteID:     noteID,
		Backend:    "obsidian",
		RemoteID:   remotePath,
		LastSyncAt: time.Now().UTC(),
		LastHash:   note.ContentHash,
		Status:     "synced",
	})

	log.Printf("imported %s from obsidian", filename)
}

func (d *Daemon) pollLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case d.syncCh <- struct{}{}:
			default:
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Daemon) syncRunner(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case <-d.syncCh:
			if _, err := d.engine.ProcessQueue(); err != nil {
				log.Printf("process queue error: %v", err)
			}
			if err := d.engine.PushAll(); err != nil {
				log.Printf("push error: %v", err)
			}
			if d.obsidianDir != "" {
				if err := d.engine.PullAll(); err != nil {
					log.Printf("pull error: %v", err)
				}
			}
			if _, err := d.engine.ProcessQueue(); err != nil {
				log.Printf("process queue error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Daemon) writePID() error {
	return os.WriteFile(d.pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0o600)
}

func (d *Daemon) readPID() (int, error) {
	data, err := os.ReadFile(d.pidPath)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid PID")
	}
	return pid, nil
}

func (d *Daemon) removePID() {
	os.Remove(d.pidPath)
}

func isNoteEvent(event fsnotify.Event) bool {
	if !strings.HasSuffix(event.Name, ".md") {
		return false
	}
	return event.Op&(fsnotify.Create|fsnotify.Write) != 0
}

func unique(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func intersect(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range b {
		set[s] = true
	}
	var result []string
	for _, s := range a {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}

func subtract(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range b {
		set[s] = true
	}
	var result []string
	for _, s := range a {
		if !set[s] {
			result = append(result, s)
		}
	}
	return result
}
