package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/ishyverma/vault-sync/internal/sync"
)

func Run(store *storage.NoteStore, engine *sync.Engine, cfg *config.Config, mgr *core.Manager) error {
	m := NewModel(store, engine, cfg, mgr)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
