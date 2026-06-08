package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ishyverma/vault-sync/internal/connectors"
	"github.com/ishyverma/vault-sync/internal/storage"
)

type Connector struct {
	repoPath      string
	autoCommit    bool
	commitMessage string
	remote        string
}

var _ connectors.Connector = (*Connector)(nil)

func NewConnector(repoPath, commitMessage, remote string, autoCommit bool) *Connector {
	return &Connector{
		repoPath:      repoPath,
		autoCommit:    autoCommit,
		commitMessage: commitMessage,
		remote:        remote,
	}
}

func (c *Connector) Name() string { return "git" }

func (c *Connector) Connect() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found: %w", err)
	}
	if _, err := os.Stat(filepath.Join(c.repoPath, ".git")); os.IsNotExist(err) {
		if err := c.exec("init"); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	}
	return nil
}

func (c *Connector) Status() (bool, error) {
	if err := c.Connect(); err != nil {
		return false, err
	}
	err := c.exec("status", "--porcelain")
	return err == nil, err
}

func (c *Connector) Push(note *storage.Note, content string, remoteID string) (string, error) {
	if !c.autoCommit {
		return filepath.Join("notes", note.Filename), nil
	}
	if err := c.Connect(); err != nil {
		return "", err
	}

	relPath := filepath.Join("notes", note.Filename)
	absPath := filepath.Join(c.repoPath, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	msg := strings.ReplaceAll(c.commitMessage, "{filename}", note.Filename)
	msg = strings.ReplaceAll(msg, "{title}", note.Title)

	if err := c.exec("add", relPath); err != nil {
		return "", fmt.Errorf("git add: %w", err)
	}
	if err := c.exec("commit", "-m", msg); err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	if c.remote != "" {
		branch := c.currentBranch()
		if err := c.exec("push", c.remote, branch); err != nil {
			return "", fmt.Errorf("git push: %w", err)
		}
	}

	return relPath, nil
}

func (c *Connector) Pull(remoteID string) (string, error) {
	if c.remote == "" {
		return "", nil
	}
	if err := c.Connect(); err != nil {
		return "", err
	}

	relPath := remoteID
	if relPath == "" {
		return "", nil
	}

	branch := c.currentBranch()
	if err := c.exec("pull", c.remote, branch); err != nil {
		return "", fmt.Errorf("git pull: %w", err)
	}

	absPath := filepath.Join(c.repoPath, relPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read pulled file: %w", err)
	}

	return string(data), nil
}

func (c *Connector) Delete(remoteID string) error {
	if !c.autoCommit {
		return nil
	}
	if err := c.Connect(); err != nil {
		return err
	}

	relPath := filepath.Join("notes", remoteID)
	absPath := filepath.Join(c.repoPath, relPath)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	msg := strings.ReplaceAll(c.commitMessage, "{filename}", remoteID)
	if err := c.exec("add", relPath); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if err := c.exec("commit", "-m", fmt.Sprintf("delete: %s", msg)); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	if c.remote != "" {
		branch := c.currentBranch()
		if err := c.exec("push", c.remote, branch); err != nil {
			return fmt.Errorf("git push: %w", err)
		}
	}

	return nil
}

func (c *Connector) currentBranch() string {
	out, err := exec.Command("git", "-C", c.repoPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "main"
	}
	return strings.TrimSpace(string(out))
}

func (c *Connector) exec(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}
