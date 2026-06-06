package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.conflicts) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "r":
		return m, m.loadConflicts()
	case "up", "k":
		if m.conflictCursor > 0 {
			m.conflictCursor--
		}
		return m, nil
	case "down", "j":
		if m.conflictCursor < len(m.conflicts)-1 {
			m.conflictCursor++
		}
		return m, nil
	case "l":
		c := m.conflicts[m.conflictCursor]
		return m, resolveConflictCmd(m.engine, c.NoteID, c.Backend, "local")
	case "R":
		c := m.conflicts[m.conflictCursor]
		return m, resolveConflictCmd(m.engine, c.NoteID, c.Backend, "remote")
	case "o":
		c := m.conflicts[m.conflictCursor]
		note, err := m.store.GetNote(c.NoteID)
		if err != nil {
			m.err = fmt.Errorf("get note: %w", err)
			return m, nil
		}
		notePath := m.manager.NotesDir() + "/" + note.Filename
		if err := openEditor(notePath); err != nil {
			m.err = fmt.Errorf("open editor: %w", err)
			return m, nil
		}
		return m, resolveConflictCmd(m.engine, c.NoteID, c.Backend, "local")
	}
	return m, nil
}

func (m model) conflictView() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Conflict Resolver"))
	b.WriteString("\n")

	if m.conflictMsg != "" {
		if strings.HasPrefix(m.conflictMsg, "✓") {
			b.WriteString(InfoStyle.Render(m.conflictMsg))
		} else {
			b.WriteString(ErrorStyle.Render(m.conflictMsg))
		}
		b.WriteString("\n\n")
	}

	if len(m.conflicts) == 0 {
		b.WriteString(InfoStyle.Render("✓ No conflicts detected"))
		return b.String()
	}

	b.WriteString(InfoStyle.Render(fmt.Sprintf("%d conflict(s) found", len(m.conflicts))))
	b.WriteString("\n\n")

	for i, c := range m.conflicts {
		prefix := "  "
		if i == m.conflictCursor {
			prefix = "▸ "
		}
		note, err := m.store.GetNote(c.NoteID)
		if err != nil {
			continue
		}

		b.WriteString(fmt.Sprintf("%s⚠ %s\n", prefix, note.Filename))
		b.WriteString(SubtleStyle.Render(fmt.Sprintf("   Backend: %s", c.Backend)))
		b.WriteString("\n")
		if c.ErrorMsg != "" {
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("   Error: %s", c.ErrorMsg)))
			b.WriteString("\n")
		}
		lastSync := "never"
		if !c.LastSyncAt.IsZero() {
			lastSync = c.LastSyncAt.Format("2006-01-02 15:04")
		}
		b.WriteString(SubtleStyle.Render(fmt.Sprintf("   Last synced: %s", lastSync)))
		b.WriteString("\n\n")
	}

	if len(m.conflicts) > 0 {
		b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)))
		b.WriteString("\n")
		b.WriteString(StatusStyle.Render("[l] Keep local  [R] Keep remote  [o] Open & edit  [r] Refresh"))
		b.WriteString("\n")
	}

	return b.String()
}
