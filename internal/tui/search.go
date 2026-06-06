package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = dashboardView
		return m, nil
	case "enter":
		if len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults) {
			note := m.searchResults[m.searchCursor]
			notePath := m.manager.NotesDir() + "/" + note.Filename
			if err := openEditor(notePath); err != nil {
				m.err = fmt.Errorf("open editor: %w", err)
			}
			return m, nil
		}
		return m, nil
	case "up", "k":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
		return m, nil
	case "down", "j":
		if m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	query := m.searchInput.Value()
	if len(query) >= 2 {
		return m, m.loadSearchResults(query)
	} else {
		m.searchResults = nil
		m.searchCursor = 0
	}

	return m, cmd
}

func (m model) searchView() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Search Notes"))
	b.WriteString("\n")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	if len(m.searchResults) > 0 {
		b.WriteString(fmt.Sprintf("Found %d results:\n\n", len(m.searchResults)))
		for i, note := range m.searchResults {
			prefix := "  "
			if i == m.searchCursor {
				prefix = "▸ "
			}
			b.WriteString(fmt.Sprintf("%s%s — %s\n", prefix, note.Filename, note.Title))
		}
	} else if len(m.searchInput.Value()) >= 2 {
		b.WriteString(SubtleStyle.Render("No results found."))
	}

	b.WriteString(StatusStyle.Render(fmt.Sprintf("\n%d notes matched", len(m.searchResults))))

	return b.String()
}
