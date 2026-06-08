package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ishyverma/vault-sync/internal/storage"
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
			if err := openEditor(notePath, m.config.Vault.Editor); err != nil {
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

	query := strings.ToLower(m.searchInput.Value())

	if len(m.searchResults) > 0 {
		b.WriteString(fmt.Sprintf("Found %d results:\n\n", len(m.searchResults)))
		for i, note := range m.searchResults {
			prefix := "  "
			if i == m.searchCursor {
				prefix = "▸ "
			}
			b.WriteString(fmt.Sprintf("%s%s — %s\n", prefix, note.Filename, note.Title))
			snippet := m.searchSnippet(note, query)
			if snippet != "" {
				b.WriteString(SubtleStyle.Render(fmt.Sprintf("   %s\n", snippet)))
			}
		}
	} else if len(query) >= 2 {
		b.WriteString(SubtleStyle.Render("No results found."))
	}

	b.WriteString(StatusStyle.Render(fmt.Sprintf("\n%d notes matched", len(m.searchResults))))

	return b.String()
}

func (m model) searchSnippet(note *storage.Note, query string) string {
	notePath := filepath.Join(m.manager.NotesDir(), note.Filename)
	data, err := os.ReadFile(notePath)
	if err != nil {
		return ""
	}
	content := string(data)

	idx := strings.Index(strings.ToLower(content), query)
	if idx < 0 {
		return ""
	}

	// Find line boundaries around the match
	start := idx
	for start > 0 && content[start] != '\n' {
		start--
	}
	if start > 0 {
		start++ // skip the newline
	}

	end := idx + len(query)
	for end < len(content) && content[end] != '\n' {
		end++
	}

	// Try to show 2-3 lines before and after
	ctxStart := start
	linesBefore := 0
	for ctxStart > 0 && linesBefore < 2 {
		ctxStart--
		if content[ctxStart] == '\n' {
			linesBefore++
		}
	}
	if ctxStart > 0 {
		ctxStart++
	}

	ctxEnd := end
	linesAfter := 0
	for ctxEnd < len(content) && linesAfter < 1 {
		if content[ctxEnd] == '\n' {
			linesAfter++
		}
		ctxEnd++
	}

	snippet := strings.TrimSpace(content[ctxStart:ctxEnd])
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	if len(snippet) > 100 {
		snippet = snippet[:100] + "…"
	}
	return snippet
}
