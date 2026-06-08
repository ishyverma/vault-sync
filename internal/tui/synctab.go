package tui

import (
	"fmt"
	"strings"
	"time"

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

	// Backend Status
	b.WriteString(TitleStyle.Render("Backend Status"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	conns := m.engine.Connectors()
	if len(conns) == 0 {
		b.WriteString(StatusStyle.Render("  No connectors configured."))
		b.WriteString("\n")
		b.WriteString(InfoStyle.Render("  Run: vault connect obsidian <path>"))
	} else {
		for name, conn := range conns {
			var healthy bool
			var connErr error
			if cached, ok := m.connHealthCache[name]; ok && time.Since(cached.checked) < 10*time.Second {
				healthy = cached.healthy
				connErr = cached.err
			} else {
				healthy, connErr = conn.Status()
			}
			statusIcon := "○"
			statusColor := SubtleStyle
			statusText := "disconnected"
			if healthy {
				statusIcon = "✓"
				statusColor = InfoStyle
				statusText = "healthy"
			} else if connErr != nil {
				statusIcon = "✗"
				statusColor = ErrorStyle
				statusText = fmt.Sprintf("error: %v", connErr)
			}
			b.WriteString(statusColor.Render(fmt.Sprintf("  %s %s", statusIcon, titleCase(name))))
			b.WriteString("\n")
			b.WriteString(SubtleStyle.Render(fmt.Sprintf("    Status: %s", statusText)))
			b.WriteString("\n\n")
		}
	}

	// Connector Sync States
	if len(m.syncStates) > 0 {
		b.WriteString(TitleStyle.Render("Per-Connector Sync State"))
		b.WriteString("\n")
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
		b.WriteString(TitleStyle.Render("Recent Sync History"))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", 40))
		b.WriteString("\n")
		for i, h := range m.syncHistory {
			if i >= 10 {
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
			case "resolved_local", "resolved_remote":
				statusIcon = "→"
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
			b.WriteString(fmt.Sprintf("  %s %s  %s → %s (%s)\n",
				statusColor.Render(statusIcon),
				SubtleStyle.Render(syncAt),
				shortID,
				h.Backend,
				h.Status,
			))
		}
	}

	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)))
	b.WriteString("\n")
	b.WriteString(StatusStyle.Render("[f] Force sync all  [r] Refresh"))

	return b.String()
}
