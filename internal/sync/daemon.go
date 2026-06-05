package sync

import (
	"context"
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
)

type Daemon struct {
	engine   *Engine
	notesDir string
	pidPath  string
	interval time.Duration
	watcher  *fsnotify.Watcher
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewDaemon(engine *Engine, notesDir string, interval time.Duration) *Daemon {
	return &Daemon{
		engine:   engine,
		notesDir: notesDir,
		pidPath:  filepath.Join(filepath.Dir(notesDir), "vaultd.pid"),
		interval: interval,
	}
}

func (d *Daemon) Start() error {
	if err := d.writePID(); err != nil {
		return fmt.Errorf("write PID: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	d.watcher = watcher
	defer watcher.Close()

	if err := watcher.Add(d.notesDir); err != nil {
		return fmt.Errorf("watch notes dir: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("vaultd started — watching %s (poll interval: %v)", d.notesDir, d.interval)

	d.wg.Add(2)
	go d.watchLoop(ctx, watcher)
	go d.pollLoop(ctx)

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

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !isNoteEvent(event) {
				continue
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(500 * time.Millisecond)
			debounceCh = debounceTimer.C

		case <-debounceCh:
			d.syncAllNotes()
			debounceTimer = nil
			debounceCh = nil

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)

		case <-ctx.Done():
			return
		}
	}
}

func (d *Daemon) pollLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.syncAllNotes()
		case <-ctx.Done():
			return
		}
	}
}

func (d *Daemon) syncAllNotes() {
	if err := d.engine.SyncAll(); err != nil {
		log.Printf("sync error: %v", err)
	}
}

func (d *Daemon) writePID() error {
	return os.WriteFile(d.pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644)
}

func (d *Daemon) readPID() (int, error) {
	data, err := os.ReadFile(d.pidPath)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
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
