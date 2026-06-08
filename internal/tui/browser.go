package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuin/goldmark"

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
			m.browserDirty = true
			m.buildBrowserCache()
			return m, nil
		case "enter":
			m.browserFiltering = false
			m.browserFilter.Blur()
			m.browserDirty = true
			m.buildBrowserCache()
			return m, nil
		}
		var filterCmd tea.Cmd
		m.browserFilter, filterCmd = m.browserFilter.Update(msg)
		m.browserDirty = true
		m.buildBrowserCache()
		return m, filterCmd
	}

	// Viewport scroll keys when preview is active
	if m.previewNote != nil {
		switch msg.String() {
		case "pgup":
			m.notePreview.HalfViewUp()
			return m, nil
		case "pgdown":
			m.notePreview.HalfViewDown()
			return m, nil
		case "g":
			m.notePreview.GotoTop()
			return m, nil
		case "G":
			m.notePreview.GotoBottom()
			return m, nil
		}
	}

	m.buildBrowserCache()
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
			if err := openEditor(notePath, m.config.Vault.Editor); err != nil {
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
		m.browserDirty = true
		return m, nil
	case "p":
		if m.previewNote != nil {
			m.previewNote.Pinned = !m.previewNote.Pinned
			if err := m.store.UpdateNote(m.previewNote); err != nil {
				m.err = fmt.Errorf("toggle pin: %w", err)
			}
			return m, m.loadNotes()
		}
	case "P":
		m.showPinnedOnly = !m.showPinnedOnly
		m.browserDirty = true
		return m, nil
	case "d":
		if m.previewNote != nil {
			name := m.previewNote.Filename
			if err := m.manager.DeleteNote(name); err != nil {
				m.err = fmt.Errorf("delete note: %w", err)
				return m, nil
			}
			m.previewNote = nil
			m.previewContent = ""
			m.notification = fmt.Sprintf("Deleted %s", name)
			m.notifUntil = time.Now().Add(3 * time.Second)
			return m, m.loadNotes()
		}
	case "t":
		if m.previewNote != nil {
			m.promptInput.Placeholder = "Enter tag name..."
			m.promptInput.SetValue("")
			m.promptActive = true
			m.promptAction = "tag"
			m.promptNoteID = m.previewNote.ID
			m.promptTitle = "Add tag"
			m.promptInput.Focus()
			return m, nil
		}
	case "a":
		if m.previewNote != nil {
			name := m.previewNote.Filename
			m.previewNote.Archived = true
			if err := m.store.UpdateNote(m.previewNote); err != nil {
				m.err = fmt.Errorf("archive note: %w", err)
				return m, nil
			}
			m.previewNote = nil
			m.previewContent = ""
			m.notification = fmt.Sprintf("Archived %s", name)
			m.notifUntil = time.Now().Add(3 * time.Second)
			return m, m.loadNotes()
		}
	case "R":
		if m.previewNote != nil {
			oldName := strings.TrimSuffix(m.previewNote.Filename, ".md")
			m.promptInput.Placeholder = "New name..."
			m.promptInput.SetValue(oldName)
			m.promptActive = true
			m.promptAction = "rename"
			m.promptNoteID = m.previewNote.ID
			m.promptTitle = "Rename note"
			m.promptInput.Focus()
			return m, nil
		}
	case "e":
		if m.previewNote != nil {
			return m, m.exportNote(m.previewNote)
		}
	case "m":
		if m.previewNote != nil {
			m.promptInput.Placeholder = "Folder name..."
			m.promptInput.SetValue("")
			m.promptActive = true
			m.promptAction = "move"
			m.promptNoteID = m.previewNote.ID
			m.promptTitle = "Move note"
			m.promptInput.Focus()
			return m, nil
		}
	}

	return m, cmd
}

func openEditor(path, editor string) error {
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
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

	if m.promptActive {
		b.WriteString(m.renderBrowserPrompt())
		return b.String()
	}

	sortLabels := []string{"name", "modified", "words"}
	b.WriteString(TitleStyle.Render(fmt.Sprintf("Note Browser — %d notes  sort: %s", len(m.notes), sortLabels[m.browserSort])))
	b.WriteString("\n")

	if m.browserFiltering {
		b.WriteString(m.browserFilter.View())
		b.WriteString("\n")
	}

	if m.browserDirty {
		m.buildBrowserCache()
	}
	b.WriteString(TableStyle.Render(m.browserTable.View()))

	b.WriteString(TableStyle.Render(m.browserTable.View()))
	b.WriteString("\n")

	b.WriteString(StatusStyle.Render("[s] Sort  [/] Filter  [o] Open  [r] Refresh  [d] Delete  [t] Tag  [a] Archive  [R] Rename  [e] Export  [m] Move"))
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

func (m *model) buildBrowserCache() {
	var rows []table.Row
	filter := strings.ToLower(m.browserFilter.Value())
	for _, n := range m.notes {
		if m.showPinnedOnly && !n.Pinned {
			continue
		}
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

		name := n.Filename
		if n.Pinned {
			name = "📌 " + name
		}

		tagsStr := ""
		if len(n.Tags) > 0 {
			limit := 3
			if len(n.Tags) < limit {
				limit = len(n.Tags)
			}
			tagsStr = strings.Join(n.Tags[:limit], ",")
		}

		syncStr := "-"
		if states, ok := m.syncStateMap[n.ID]; ok {
			allSynced := true
			hasAny := false
			for _, s := range states {
				hasAny = true
				if s.Status != "synced" {
					allSynced = false
					syncStr = s.Status
					break
				}
			}
			if hasAny && allSynced {
				syncStr = "synced"
			} else if !hasAny {
				syncStr = "-"
			}
		} else {
			syncStr = "local"
		}

		rows = append(rows, table.Row{
			name,
			truncate(n.Title, 24),
			fmt.Sprintf("%d", n.WordCount),
			modified,
			truncate(tagsStr, 13),
			syncStr,
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

	m.browserRowsCache = rows
	m.browserTable.SetRows(rows)
	m.browserDirty = false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func (m *model) exportNote(note *storage.Note) tea.Cmd {
	return func() tea.Msg {
		notePath := filepath.Join(m.manager.NotesDir(), note.Filename)
		data, err := os.ReadFile(notePath)
		if err != nil {
			return errMsg{fmt.Errorf("read note: %w", err)}
		}

		htmlName := strings.TrimSuffix(note.Filename, ".md") + ".html"
		htmlPath := filepath.Join(m.manager.NotesDir(), htmlName)

		var body strings.Builder
		body.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
		body.WriteString(fmt.Sprintf("<title>%s</title>\n", note.Title))
		body.WriteString("<style>body{max-width:800px;margin:40px auto;padding:0 20px;font-family:system-ui,-apple-system,sans-serif;line-height:1.6}pre{background:#f5f5f5;padding:1em;overflow-x:auto}code{background:#f0f0f0;padding:.2em .4em}</style>\n")
		body.WriteString("</head>\n<body>\n")

		var buf strings.Builder
		md := goldmark.New()
		if err := md.Convert(data, &buf); err != nil {
			body.WriteString("<pre>" + string(data) + "</pre>\n")
		} else {
			body.WriteString(buf.String())
		}

		body.WriteString("</body>\n</html>\n")

		if err := os.WriteFile(htmlPath, []byte(body.String()), 0o644); err != nil {
			return errMsg{fmt.Errorf("write html: %w", err)}
		}

		return exportDoneMsg{name: htmlName}
	}
}

type exportDoneMsg struct {
	name string
}

func (m model) renderBrowserPrompt() string {
	if !m.promptActive {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(TitleStyle.Render(m.promptTitle))
	b.WriteString("\n")
	b.WriteString(m.promptInput.View())
	b.WriteString("\n")
	b.WriteString(StatusStyle.Render("[enter] Confirm  [esc] Cancel"))
	return b.String()
}
