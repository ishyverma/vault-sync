package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/ishyverma/vault-sync/internal/sync"
)

type viewState int

const (
	dashboardView viewState = iota
	browserView
	searchView
	syncView
	settingsView
	conflictView
)

type model struct {
	state viewState

	store   *storage.NoteStore
	engine  *sync.Engine
	config  *config.Config
	manager *core.Manager

	help       help.Model
	keys       keyMap
	width      int
	height     int
	err        error

	notes      []*storage.Note
	searchInput textinput.Model
	searchResults []*storage.Note
	searchCursor int

	browserFilter    textinput.Model
	browserFiltering bool
	browserSort      int // 0=name, 1=modified, 2=words
	browserTable     table.Model
	notePreview      viewport.Model
	previewNote      *storage.Note
	previewContent   string

	syncHistory []*storage.SyncHistoryEntry
	syncStates  []*storage.SyncState

	conflicts []*storage.SyncState
	conflictDetail string
	conflictCursor int
	conflictMsg string
}

func NewModel(store *storage.NoteStore, engine *sync.Engine, cfg *config.Config, mgr *core.Manager) model {
	ti := textinput.New()
	ti.Placeholder = "Search notes..."
	ti.CharLimit = 100
	ti.Width = 40
	ti.Focus()

	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Name", Width: 20},
			{Title: "Title", Width: 25},
			{Title: "Words", Width: 6},
			{Title: "Modified", Width: 12},
		}),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFF")).
		Background(highlight).
		Bold(false)
	t.SetStyles(s)

	bf := textinput.New()
	bf.Placeholder = "Filter notes..."
	bf.CharLimit = 50
	bf.Width = 30

	vp := viewport.New(70, 10)
	vp.Style = lipgloss.NewStyle().PaddingLeft(2)

	return model{
		state:        dashboardView,
		store:        store,
		engine:       engine,
		config:       cfg,
		manager:      mgr,
		help:         help.New(),
		keys:         defaultKeyMap(),
		searchInput:    ti,
		browserFilter:  bf,
		browserTable:   t,
		notePreview:    vp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadNotes(),
		m.loadSyncData(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		tableWidth := msg.Width - 10
		if tableWidth < 40 {
			tableWidth = 40
		}
		curCols := m.browserTable.Columns()
		if len(curCols) >= 4 {
			nameW := tableWidth * 3 / 10
			titleW := tableWidth * 4 / 10
			wordsW := 6
			dateW := tableWidth - nameW - titleW - wordsW - 3
			if dateW < 10 {
				dateW = 10
			}
			curCols[0].Width = nameW
			curCols[1].Width = titleW
			curCols[2].Width = wordsW
			curCols[3].Width = dateW
			m.browserTable.SetColumns(curCols)
		}
		m.notePreview.Width = msg.Width - 10
		m.notePreview.Height = msg.Height / 3
		return m, nil

	case notesLoadedMsg:
		m.notes = msg.notes
		m.browserTable.SetRows(m.buildTableRows())
		return m, nil

	case syncDataMsg:
		m.syncStates = msg.states
		return m, nil

	case searchResultsMsg:
		m.searchResults = msg.results
		return m, nil

	case conflictsLoadedMsg:
		m.conflicts = msg.states
		m.conflictCursor = 0
		return m, nil

	case syncCompleteMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("sync failed: %w", msg.err)
		}
		return m, tea.Batch(m.loadNotes(), m.loadSyncData())

	case conflictResolvedMsg:
		if msg.err != nil {
			m.conflictMsg = fmt.Sprintf("✗ Failed to resolve %s/%s: %v", msg.noteID, msg.backend, msg.err)
		} else {
			m.conflictMsg = fmt.Sprintf("✓ Resolved conflict: %s/%s", msg.noteID, msg.backend)
		}
		return m, m.loadConflicts()

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.err != nil {
			m.err = nil
			return m, nil
		}

		if m.state == searchView {
			switch {
			case key.Matches(msg, m.keys.TabNext):
				return m.switchView(viewState((int(m.state) + 1) % 6))
			case key.Matches(msg, m.keys.TabPrev):
				return m.switchView(viewState((int(m.state) + 5) % 6))
			}
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			return m.updateSearch(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keys.TabNext):
			return m.switchView(viewState((int(m.state) + 1) % 6))
		case key.Matches(msg, m.keys.TabPrev):
			return m.switchView(viewState((int(m.state) + 5) % 6))
		case key.Matches(msg, m.keys.Dashboard):
			return m.switchView(dashboardView)
		case key.Matches(msg, m.keys.Browser):
			return m.switchView(browserView)
		case m.state != browserView && key.Matches(msg, m.keys.Search):
			return m.switchView(searchView)
		case key.Matches(msg, m.keys.Sync):
			return m.switchView(syncView)
		case key.Matches(msg, m.keys.Settings):
			return m.switchView(settingsView)
		case key.Matches(msg, m.keys.Conflicts):
			return m.switchView(conflictView)
		}

		switch m.state {
		case dashboardView:
			return m.updateDashboard(msg)
		case browserView:
			return m.updateBrowser(msg)
		case conflictView:
			return m.updateConflict(msg)
		case syncView:
			return m.updateSync(msg)
		}
	}

	return m, nil
}

func (m model) View() string {
	header := m.renderHeader()

	var content string
	if m.err != nil {
		content = lipgloss.JoinVertical(lipgloss.Top,
			ErrorStyle.Render("Error: "+m.err.Error()),
			SubtleStyle.Render("\nPress any key to dismiss"),
		)
	} else {
		switch m.state {
		case dashboardView:
			content = m.dashboardView()
		case browserView:
			content = m.browserView()
		case searchView:
			content = m.searchView()
		case syncView:
			content = m.syncView()
		case settingsView:
			content = m.settingsView()
		case conflictView:
			content = m.conflictView()
		}
	}

	tabs := m.renderTabs()
	help := m.renderHelp()

	main := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		content,
		tabs,
		help,
	)

	if m.width > 0 {
		return lipgloss.NewStyle().Width(m.width).Render(main)
	}
	return main
}

func (m model) renderHeader() string {
	b := strings.Builder{}
	b.WriteString(TitleStyle.Render("VaultSync"))
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)))
	return b.String()
}

func (m model) renderTabs() string {
	tabs := []struct {
		label string
		state viewState
	}{
		{"📊 Dashboard", dashboardView},
		{"📝 Notes", browserView},
		{"🔍 Search", searchView},
		{"🔄 Sync", syncView},
		{"⚙ Settings", settingsView},
		{"⚠ Conflicts", conflictView},
	}

	var rendered []string
	for _, t := range tabs {
		if t.state == m.state {
			rendered = append(rendered, ActiveTabStyle.Render(t.label))
		} else {
			rendered = append(rendered, TabStyle.Render(t.label))
		}
	}

	return lipgloss.NewStyle().
		PaddingTop(1).
		PaddingBottom(1).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, rendered...))
}

func (m model) renderHelp() string {
	return m.help.View(m.keys)
}

func (m *model) loadNotes() tea.Cmd {
	return func() tea.Msg {
		notes, err := m.store.ListNotes()
		if err != nil {
			return errMsg{err}
		}
		return notesLoadedMsg{notes}
	}
}

func (m *model) loadSyncData() tea.Cmd {
	return func() tea.Msg {
		states, err := m.engine.AllSyncStatuses()
		if err != nil {
			return errMsg{err}
		}
		return syncDataMsg{states: states}
	}
}

func (m *model) loadConflicts() tea.Cmd {
	return func() tea.Msg {
		states, err := m.store.ListSyncStatesByStatus("conflict")
		if err != nil {
			return errMsg{err}
		}
		return conflictsLoadedMsg{states}
	}
}

func (m *model) loadSearchResults(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := m.store.SearchNotes(query)
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{results}
	}
}

type errMsg struct{ err error }
type notesLoadedMsg struct{ notes []*storage.Note }
type syncDataMsg struct {
	states  []*storage.SyncState
}
type searchResultsMsg struct{ results []*storage.Note }
type conflictsLoadedMsg struct{ states []*storage.SyncState }
type syncCompleteMsg struct{ err error }
type conflictResolvedMsg struct {
	noteID  string
	backend string
	err     error
}

func syncAllCmd(engine *sync.Engine) tea.Cmd {
	return func() tea.Msg {
		err := engine.SyncAll()
		return syncCompleteMsg{err: err}
	}
}

func resolveConflictCmd(engine *sync.Engine, noteID, backend, strategy string) tea.Cmd {
	return func() tea.Msg {
		err := engine.ResolveConflict(noteID, backend, strategy)
		return conflictResolvedMsg{noteID: noteID, backend: backend, err: err}
	}
}

func (m model) switchView(target viewState) (tea.Model, tea.Cmd) {
	m.state = target
	m.err = nil
	switch target {
	case browserView:
		return m, m.loadNotes()
	case syncView:
		return m, m.loadSyncData()
	case conflictView:
		return m, m.loadConflicts()
	case searchView:
		m.searchInput.Focus()
		return m, nil
	}
	return m, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("2006-01-02 15:04")
}

func pluralize(n int, s string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, s)
	}
	return fmt.Sprintf("%d %ss", n, s)
}
