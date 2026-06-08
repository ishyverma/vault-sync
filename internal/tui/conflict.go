package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showDiff {
		switch msg.String() {
		case "esc", "q", "enter":
			m.showDiff = false
			return m, nil
		case "up", "k":
			m.diffView.LineUp(1)
			return m, nil
		case "down", "j":
			m.diffView.LineDown(1)
			return m, nil
		case "pgup":
			m.diffView.ViewUp()
			return m, nil
		case "pgdown":
			m.diffView.ViewDown()
			return m, nil
		}
		return m, nil
	}

	if len(m.conflicts) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "r":
		return m, m.loadConflicts()
	case "d":
		return m, m.loadConflictDiff()
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
	case "enter":
		return m, m.loadConflictDiff()
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
		if _, err := os.Stat(notePath); err != nil {
			m.err = fmt.Errorf("note file missing: %w", err)
			return m, nil
		}
		if err := openEditor(notePath, m.config.Vault.Editor); err != nil {
			m.err = fmt.Errorf("open editor: %w", err)
			return m, nil
		}
		return m, resolveConflictCmd(m.engine, c.NoteID, c.Backend, "local")
	}
	return m, nil
}

func (m model) conflictView() string {
	var b strings.Builder

	if m.showDiff {
		b.WriteString(TitleStyle.Render("Conflict Diff"))
		b.WriteString("\n\n")
		m.diffView.SetContent(m.diffContent)
		b.WriteString(m.diffView.View())
		b.WriteString("\n\n")
		b.WriteString(StatusStyle.Render("[esc] Back to conflict list  [j/k] Scroll"))
		return b.String()
	}

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
		filename := m.conflictNoteMap[c.NoteID]
		if filename == "" {
			filename = c.NoteID
		}

		b.WriteString(fmt.Sprintf("%s⚠ %s\n", prefix, filename))
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
		b.WriteString(StatusStyle.Render("[l] Keep local  [R] Keep remote  [d/enter] View diff  [o] Open & edit  [r] Refresh"))
		b.WriteString("\n")
	}

	return b.String()
}
