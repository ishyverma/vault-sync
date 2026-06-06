package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ishyverma/vault-sync/internal/storage"
)

type Connector struct {
	vaultPath string
	subfolder string
	notesDir  string
	wikilinks bool
	targetDir string
}

func NewConnector(vaultPath, subfolder, notesDir string, wikilinks bool) *Connector {
	expanded := expandHome(vaultPath)
	return &Connector{
		vaultPath: expanded,
		subfolder: subfolder,
		notesDir:  notesDir,
		wikilinks: wikilinks,
		targetDir: filepath.Join(expanded, subfolder),
	}
}

func (c *Connector) Name() string {
	return "obsidian"
}

func (c *Connector) Connect() error {
	return os.MkdirAll(c.targetDir, 0o755)
}

func (c *Connector) Status() (bool, error) {
	info, err := os.Stat(c.vaultPath)
	if err != nil {
		return false, fmt.Errorf("obsidian vault not accessible: %w", err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("obsidian vault path is not a directory: %s", c.vaultPath)
	}
	return true, nil
}

func (c *Connector) Push(note *storage.Note, content string, remoteID string) (string, error) {
	remotePath := c.resolvePath(note.Path)
	if remotePath == "" {
		remotePath = c.resolvePath(note.Filename)
	}

	if err := os.MkdirAll(filepath.Dir(remotePath), 0o755); err != nil {
		return "", fmt.Errorf("create obsidian subdir: %w", err)
	}

	if err := atomicWrite(remotePath, content); err != nil {
		return "", fmt.Errorf("write to obsidian: %w", err)
	}

	return remotePath, nil
}

func (c *Connector) Pull(remoteID string) (string, error) {
	data, err := os.ReadFile(remoteID)
	if err != nil {
		return "", fmt.Errorf("read from obsidian: %w", err)
	}
	return string(data), nil
}

func (c *Connector) Delete(remoteID string) error {
	if err := os.Remove(remoteID); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete from obsidian: %w", err)
	}
	return nil
}

func (c *Connector) TargetDir() string {
	return c.targetDir
}

func (c *Connector) resolvePath(notePath string) string {
	if notePath == "" {
		return ""
	}
	clean := filepath.Clean(notePath)
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, "..") {
		return ""
	}
	return filepath.Join(c.targetDir, clean)
}

func atomicWrite(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vault-sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Sync()
	tmp.Close()

	return os.Rename(tmpName, path)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
