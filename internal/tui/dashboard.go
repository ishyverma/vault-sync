package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n":
		name := fmt.Sprintf("note-%s", time.Now().Format("2006-01-02-150405"))
		note, err := m.manager.CreateNote(name, "")
		if err != nil {
			m.err = fmt.Errorf("create note: %w", err)
			return m, nil
		}
		notePath := m.manager.NotesDir() + "/" + note.Filename
		if err := openEditor(notePath, m.config.Vault.Editor); err != nil {
			m.err = fmt.Errorf("open editor: %w", err)
		}
		return m, tea.Batch(m.loadNotes(), syncAllCmd(m.engine))
	case "o":
		return m.switchView(browserView)
	case "s":
		return m, syncAllCmd(m.engine)
	}
	return m, nil
}

func (m model) dashboardView() string {
	var b strings.Builder

	totalNotes := len(m.notes)

	b.WriteString(StatusStyle.Render(fmt.Sprintf("📓 %s  |  ✍ %d words today  |  %s  |  🔥 %d day streak",
		pluralize(totalNotes, "note"), m.dashWords, m.dashStorage, m.dashStreak)))
	b.WriteString("\n\n")

	b.WriteString(m.dashSyncStr)
	b.WriteString(m.dashTagsStr)
	b.WriteString(m.dashRecentStr)
	b.WriteString(m.dashConnStr)

	// Shortcuts bar
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 50)))
	b.WriteString("\n")
	b.WriteString(StatusStyle.Render("[n] New  [o] Open  [/] Search  [s] Sync  [?] Help  [q] Quit"))

	return b.String()
}

func (m *model) buildDashboardCache() {
	m.dashStorage, m.dashStreak, m.dashWords = m.computeDashboardStats()
	m.dashSyncStr = m.buildDashSyncStr()
	m.dashTagsStr = m.buildDashTagsStr()
	m.dashRecentStr = m.buildDashRecentStr()
	m.dashConnStr = m.buildDashConnStr()
}

func (m model) computeDashboardStats() (string, int, int) {
	wordsToday := 0
	today := time.Now().Format("2006-01-02")
	for _, n := range m.notes {
		if !n.ModifiedAt.IsZero() && n.ModifiedAt.Format("2006-01-02") == today {
			wordsToday += n.WordCount
		}
	}

	var storageUsed int64
	notesDir := m.manager.NotesDir()
	for _, n := range m.notes {
		fi, err := os.Stat(filepath.Join(notesDir, n.Filename))
		if err == nil {
			storageUsed += fi.Size()
		}
	}
	storageStr := formatBytes(storageUsed)

	streak := 0
	check := time.Now()
	noteDateSet := make(map[string]bool)
	for _, n := range m.notes {
		if !n.ModifiedAt.IsZero() {
			noteDateSet[n.ModifiedAt.Format("2006-01-02")] = true
		}
	}
	for i := 0; i < 365; i++ {
		day := check.Format("2006-01-02")
		if noteDateSet[day] {
			streak++
			check = check.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	return storageStr, streak, wordsToday
}

func (m model) buildDashSyncStr() string {
	var b strings.Builder
	synced := 0
	localOnly := 0
	failed := 0
	var latestSync time.Time
	for _, s := range m.syncStates {
		switch s.Status {
		case "synced":
			synced++
		case "local_only":
			localOnly++
		case "failed":
			failed++
		}
		if s.LastSyncAt.After(latestSync) {
			latestSync = s.LastSyncAt
		}
	}
	lastSync := "never"
	if !latestSync.IsZero() {
		lastSync = latestSync.Format("2006-01-02 15:04")
	}

	b.WriteString(TitleStyle.Render("Sync Status"))
	b.WriteString("\n")
	b.WriteString(InfoStyle.Render(fmt.Sprintf("  ✓ Synced:     %d", synced)))
	b.WriteString("\n")
	if localOnly > 0 {
		b.WriteString(SubtleStyle.Render(fmt.Sprintf("  ○ Local only: %d", localOnly)))
		b.WriteString("\n")
	}
	if failed > 0 {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("  ✗ Failed:     %d", failed)))
		b.WriteString("\n")
	}
	b.WriteString(SubtleStyle.Render(fmt.Sprintf("  Last sync: %s", lastSync)))
	if m.syncQueueLen > 0 {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("  ⏳ Pending:    %d jobs", m.syncQueueLen)))
	}
	b.WriteString("\n\n")
	return b.String()
}

func (m model) buildDashTagsStr() string {
	var b strings.Builder
	tagCounts := map[string]int{}
	for _, n := range m.notes {
		for _, tag := range n.Tags {
			tagCounts[tag]++
		}
	}
	if len(tagCounts) > 0 {
		type tagCount struct {
			tag   string
			count int
		}
		sorted := make([]tagCount, 0, len(tagCounts))
		for t, c := range tagCounts {
			sorted = append(sorted, tagCount{t, c})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
		if len(sorted) > 5 {
			sorted = sorted[:5]
		}
		b.WriteString(TitleStyle.Render("Top Tags"))
		b.WriteString("\n")
		for _, tc := range sorted {
			b.WriteString(fmt.Sprintf("  #%s (%d)\n", tc.tag, tc.count))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) buildDashRecentStr() string {
	if len(m.notes) == 0 {
		return ""
	}
	var b strings.Builder
	recent := m.notes
	if len(recent) > 5 {
		recent = recent[:5]
	}
	b.WriteString(TitleStyle.Render("Recent Notes"))
	b.WriteString("\n")
	for _, n := range recent {
		modified := "unknown"
		if !n.ModifiedAt.IsZero() {
			modified = n.ModifiedAt.Format("2006-01-02")
		}
		b.WriteString(fmt.Sprintf("  %s — %s (%s)\n", n.Filename, n.Title, modified))
	}
	return b.String()
}

func (m *model) buildDashConnStr() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Connections"))
	b.WriteString("\n")
	conns := m.engine.Connectors()
	if len(conns) == 0 {
		b.WriteString(SubtleStyle.Render("  No connectors configured"))
		b.WriteString("\n")
		return b.String()
	}
	backends := map[string]bool{
		"notion":   m.config.Backends.Notion.Enabled,
		"obsidian": m.config.Backends.Obsidian.Enabled,
		"git":      m.config.Backends.Git.Enabled,
	}
	for name, enabled := range backends {
		if !enabled {
			b.WriteString(fmt.Sprintf("  ○ %s - not configured\n", titleCase(name)))
			continue
		}
		if conn, ok := conns[name]; ok {
			healthy, err := m.cachedConnHealth(name, conn)
			if healthy {
				b.WriteString(InfoStyle.Render(fmt.Sprintf("  ● %s - healthy", titleCase(name))))
			} else if err != nil {
				b.WriteString(ErrorStyle.Render(fmt.Sprintf("  ● %s - error: %v", titleCase(name), err)))
			} else {
				b.WriteString(ErrorStyle.Render(fmt.Sprintf("  ● %s - unhealthy", titleCase(name))))
			}
		} else {
			b.WriteString(SubtleStyle.Render(fmt.Sprintf("  ○ %s - disconnected", titleCase(name))))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m *model) cachedConnHealth(name string, conn interface{ Status() (bool, error) }) (bool, error) {
	if cached, ok := m.connHealthCache[name]; ok && time.Since(cached.checked) < 10*time.Second {
		return cached.healthy, cached.err
	}
	healthy, err := conn.Status()
	m.connHealthCache[name] = connHealthResult{healthy: healthy, err: err, checked: time.Now()}
	return healthy, err
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
