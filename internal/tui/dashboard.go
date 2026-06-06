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
		if err := openEditor(notePath); err != nil {
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

	queueLen, _ := m.engine.QueueLength()

	wordsToday := 0
	today := time.Now().Format("2006-01-02")
	for _, n := range m.notes {
		if !n.ModifiedAt.IsZero() && n.ModifiedAt.Format("2006-01-02") == today {
			wordsToday += n.WordCount
		}
	}

	// Storage used
	var storageUsed int64
	notesDir := m.manager.NotesDir()
	for _, n := range m.notes {
		fi, err := os.Stat(filepath.Join(notesDir, n.Filename))
		if err == nil {
			storageUsed += fi.Size()
		}
	}
	storageStr := formatBytes(storageUsed)

	// Writing streak
	streak := 0
	check := time.Now()
	for i := 0; i < 365; i++ {
		day := check.Format("2006-01-02")
		hasNote := false
		for _, n := range m.notes {
			if !n.ModifiedAt.IsZero() && n.ModifiedAt.Format("2006-01-02") == day {
				hasNote = true
				break
			}
		}
		if hasNote {
			streak++
			check = check.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	b.WriteString(StatusStyle.Render(fmt.Sprintf("📓 %s  |  ✍ %d words today  |  %s  |  🔥 %d day streak",
		pluralize(totalNotes, "note"), wordsToday, storageStr, streak)))
	b.WriteString("\n\n")

	// Sync status block
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
	if queueLen > 0 {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("  ⏳ Pending:    %d jobs", queueLen)))
	}
	b.WriteString("\n\n")

	// Top tags
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

	// Recent notes
	recent := m.notes
	if len(recent) > 5 {
		recent = recent[:5]
	}
	if len(recent) > 0 {
		b.WriteString(TitleStyle.Render("Recent Notes"))
		b.WriteString("\n")
		for _, n := range recent {
			modified := "unknown"
			if !n.ModifiedAt.IsZero() {
				modified = n.ModifiedAt.Format("2006-01-02")
			}
			b.WriteString(fmt.Sprintf("  %s — %s (%s)\n", n.Filename, n.Title, modified))
		}
	}

	// Shortcuts bar
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 50)))
	b.WriteString("\n")
	b.WriteString(StatusStyle.Render("[n] New  [o] Open  [/] Search  [s] Sync  [?] Help  [q] Quit"))

	return b.String()
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
