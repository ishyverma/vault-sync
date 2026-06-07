package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) updateSync(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		return m, m.loadSyncData()
	case "f":
		return m, syncAllCmd(m.engine)
	}
	return m, nil
}

func (m model) syncView() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Sync Monitor"))
	b.WriteString("\n")

	if m.syncQueueLen > 0 {
		b.WriteString(fmt.Sprintf("Queue: %d pending\n", m.syncQueueLen))
		b.WriteString("\n")
	}

	if len(m.syncStates) == 0 {
		b.WriteString(StatusStyle.Render("No connectors configured."))
		b.WriteString("\n")
		b.WriteString(InfoStyle.Render("Run: vault connect obsidian <path>"))
	} else {
		b.WriteString("Connector Status\n")
		b.WriteString(strings.Repeat("─", 40))
		b.WriteString("\n")

		for _, s := range m.syncStates {
			statusColor := SubtleStyle
			icon := "○"
			switch s.Status {
			case "synced":
				statusColor = InfoStyle
				icon = "✓"
			case "failed":
				statusColor = ErrorStyle
				icon = "✗"
			case "conflict":
				statusColor = lipgloss.NewStyle().Foreground(warning)
				icon = "⚠"
			}

			lastSync := "never"
			if !s.LastSyncAt.IsZero() {
				lastSync = s.LastSyncAt.Format("2006-01-02 15:04")
			}

			b.WriteString(statusColor.Render(fmt.Sprintf("  %s %s", icon, s.Backend)))
			b.WriteString("\n")
			b.WriteString(SubtleStyle.Render(fmt.Sprintf("    Status:   %s", s.Status)))
			b.WriteString("\n")
			b.WriteString(SubtleStyle.Render(fmt.Sprintf("    Last sync: %s", lastSync)))
			b.WriteString("\n")
			if s.ErrorMsg != "" {
				b.WriteString(ErrorStyle.Render(fmt.Sprintf("    Error: %s", s.ErrorMsg)))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	if len(m.syncHistory) > 0 {
		b.WriteString(strings.Repeat("─", 40))
		b.WriteString("\n")
		b.WriteString("Recent Sync History\n")
		b.WriteString(strings.Repeat("─", 40))
		b.WriteString("\n")
		for i, h := range m.syncHistory {
			if i >= 8 {
				break
			}
			statusIcon := "✓"
			statusColor := InfoStyle
			switch h.Status {
			case "failed", "conflict":
				statusIcon = "✗"
				statusColor = ErrorStyle
			case "pending":
				statusIcon = "⟳"
				statusColor = StatusStyle
			}
			syncAt := "?"
			if !h.SyncedAt.IsZero() {
				syncAt = h.SyncedAt.Format("15:04:05")
			}
			shortID := h.NoteID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			b.WriteString(fmt.Sprintf("  %s %s  %s → %s\n",
				statusColor.Render(statusIcon),
				SubtleStyle.Render(syncAt),
				shortID,
				h.Backend,
			))
		}
	}

	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)))
	b.WriteString("\n")
	b.WriteString(StatusStyle.Render("[f] Force sync all  [r] Refresh"))

	return b.String()
}
