package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
)

func (m model) updateBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.browserFiltering {
		switch msg.String() {
		case "esc":
			m.browserFiltering = false
			m.browserFilter.SetValue("")
			m.browserFilter.Blur()
			return m, nil
		case "enter":
			m.browserFiltering = false
			m.browserFilter.Blur()
			return m, nil
		}
		var filterCmd tea.Cmd
		m.browserFilter, filterCmd = m.browserFilter.Update(msg)
		m.browserTable.SetRows(m.buildTableRows())
		return m, filterCmd
	}

	// Viewport scroll keys when preview is active
	if m.previewNote != nil {
		switch msg.String() {
		case "pgup":
			m.notePreview.HalfViewUp()
		case "pgdown":
			m.notePreview.HalfViewDown()
		case "g":
			m.notePreview.GotoTop()
		case "G":
			m.notePreview.GotoBottom()
		}
	}

	m.browserTable.SetRows(m.buildTableRows())
	m.browserTable, cmd = m.browserTable.Update(msg)

	if row := m.browserTable.SelectedRow(); len(row) > 0 {
		filename := row[0]
		for _, n := range m.notes {
			if n.Filename == filename {
				m.previewNote = n
				m.loadNotePreview(n)
				break
			}
		}
	}

	switch msg.String() {
	case "o":
		if m.previewNote != nil {
			notePath := m.manager.NotesDir() + "/" + m.previewNote.Filename
			m.manager.SyncFromDisk(m.previewNote.ID)
			if err := openEditor(notePath); err != nil {
				m.err = fmt.Errorf("open editor: %w", err)
			}
			return m, m.loadNotes()
		}
	case "r":
		return m, m.loadNotes()
	case "/":
		m.browserFiltering = true
		m.browserFilter.Focus()
		return m, nil
	case "s":
		m.browserSort = (m.browserSort + 1) % 3
		return m, nil
	}

	return m, cmd
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m model) loadNotePreview(note *storage.Note) {
	notePath := filepath.Join(m.manager.NotesDir(), note.Filename)
	rendered := m.renderNotePreview(notePath)
	m.notePreview.SetContent(rendered)
}

func (m model) renderNotePreview(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading note: %v", err)
	}

	content := string(data)
	_, body, parseErr := core.ParseFrontmatter(content)
	if parseErr == nil && body != "" {
		content = body
	}

	truncated := content
	if len(truncated) > 2000 {
		truncated = truncated[:2000] + "\n\n... (truncated)"
	}

	style := "dark"
	if !lipgloss.HasDarkBackground() {
		style = "light"
	}
	rendered, err := glamour.Render(truncated, style)
	if err != nil {
		return truncated
	}
	return rendered
}

func (m model) browserView() string {
	var b strings.Builder

	sortLabels := []string{"name", "modified", "words"}
	b.WriteString(TitleStyle.Render(fmt.Sprintf("Note Browser — %d notes  sort: %s", len(m.notes), sortLabels[m.browserSort])))
	b.WriteString("\n")

	if m.browserFiltering {
		b.WriteString(m.browserFilter.View())
		b.WriteString("\n")
	}

	rows := m.buildTableRows()
	m.browserTable.SetRows(rows)

	b.WriteString(TableStyle.Render(m.browserTable.View()))
	b.WriteString("\n")

	b.WriteString(StatusStyle.Render("[s] Sort  [/] Filter  [o] Open  [r] Refresh"))
	b.WriteString("\n")

	if m.previewNote != nil {
		b.WriteString(DividerStyle.Render(strings.Repeat("─", 60)))
		b.WriteString("\n")
		b.WriteString(StatusStyle.Render(fmt.Sprintf("  %s — %d words — Tags: %s",
			m.previewNote.Filename, m.previewNote.WordCount, strings.Join(m.previewNote.Tags, ", "))))
		b.WriteString("\n")
		b.WriteString(m.notePreview.View())
	}

	return b.String()
}

func (m model) buildTableRows() []table.Row {
	var rows []table.Row
	filter := strings.ToLower(m.browserFilter.Value())
	for _, n := range m.notes {
		if filter != "" {
			name := strings.ToLower(n.Filename)
			title := strings.ToLower(n.Title)
			if !strings.Contains(name, filter) && !strings.Contains(title, filter) {
				continue
			}
		}
		modified := "unknown"
		if !n.ModifiedAt.IsZero() {
			modified = n.ModifiedAt.Format("2006-01-02")
		}
		rows = append(rows, table.Row{
			n.Filename,
			truncate(n.Title, 24),
			fmt.Sprintf("%d", n.WordCount),
			modified,
		})
	}

	switch m.browserSort {
	case 1: // by modified (desc)
		sort.SliceStable(rows, func(i, j int) bool { return rows[i][3] > rows[j][3] })
	case 2: // by words (desc)
		sort.SliceStable(rows, func(i, j int) bool {
			wi, _ := strconv.Atoi(rows[i][2])
			wj, _ := strconv.Atoi(rows[j][2])
			return wi > wj
		})
	}

	return rows
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
