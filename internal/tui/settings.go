package tui

import (
	"fmt"
	"strings"
)

func (m model) settingsView() string {
	var b strings.Builder

	if m.promptActive && m.promptAction == "editor" {
		b.WriteString(TitleStyle.Render("Change Editor"))
		b.WriteString("\n")
		b.WriteString(m.promptInput.View())
		b.WriteString("\n")
		b.WriteString(StatusStyle.Render("[enter] Confirm  [esc] Cancel"))
		return b.String()
	}

	b.WriteString(TitleStyle.Render("Settings"))
	b.WriteString("\n")

	b.WriteString("Vault\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Path:   %s\n", m.config.Vault.Path))
	b.WriteString(fmt.Sprintf("  Editor: %s\n", m.config.Vault.Editor))
	if m.config.Vault.TemplateDir != "" {
		b.WriteString(fmt.Sprintf("  Templates: %s\n", m.config.Vault.TemplateDir))
	}
	b.WriteString("\n")

	b.WriteString("Obsidian\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	if m.config.Backends.Obsidian.Enabled {
		b.WriteString(fmt.Sprintf("  Vault path: %s\n", m.config.Backends.Obsidian.VaultPath))
		b.WriteString(fmt.Sprintf("  Subfolder:  %s\n", m.config.Backends.Obsidian.Subfolder))
		b.WriteString(fmt.Sprintf("  Wiki links: %v\n", m.config.Backends.Obsidian.Wikilinks))
	} else {
		b.WriteString("  Not configured\n")
	}
	b.WriteString("\n")

	b.WriteString("Notion\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	if m.config.Backends.Notion.Enabled {
		b.WriteString(fmt.Sprintf("  Target page: %s\n", m.config.Backends.Notion.TargetPageID))
		b.WriteString(fmt.Sprintf("  Database ID: %s\n", m.config.Backends.Notion.DatabaseID))
	} else {
		b.WriteString("  Not configured\n")
	}
	b.WriteString("\n")

	b.WriteString("Git\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	if m.config.Backends.Git.Enabled {
		b.WriteString(fmt.Sprintf("  Repo path: %s\n", m.config.Backends.Git.RepoPath))
		b.WriteString(fmt.Sprintf("  Auto-commit: %v\n", m.config.Backends.Git.AutoCommit))
		b.WriteString(fmt.Sprintf("  Remote: %s\n", m.config.Backends.Git.Remote))
	} else {
		b.WriteString("  Not configured\n")
	}
	b.WriteString("\n")

	b.WriteString("Sync\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Auto-sync:  %v\n", m.config.Sync.AutoSync))
	b.WriteString(fmt.Sprintf("  Interval:   %ds\n", m.config.Sync.SyncInterval))
	b.WriteString(fmt.Sprintf("  Strategy:   %s\n", m.config.Sync.ConflictStrategy))
	b.WriteString(fmt.Sprintf("  Retry limit: %d\n", m.config.Sync.QueueRetryLimit))
	b.WriteString("\n")

	b.WriteString("Hooks\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	if m.config.Hooks.PreSync != "" {
		b.WriteString(fmt.Sprintf("  Pre-sync:  %s\n", m.config.Hooks.PreSync))
	}
	if m.config.Hooks.PostSync != "" {
		b.WriteString(fmt.Sprintf("  Post-sync: %s\n", m.config.Hooks.PostSync))
	}
	if m.config.Hooks.OnConflict != "" {
		b.WriteString(fmt.Sprintf("  On conflict: %s\n", m.config.Hooks.OnConflict))
	}
	if m.config.Hooks.PreSync == "" && m.config.Hooks.PostSync == "" && m.config.Hooks.OnConflict == "" {
		b.WriteString("  (none configured)\n")
	}
	b.WriteString("\n")

	b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)))
	b.WriteString("\n")
	b.WriteString(InfoStyle.Render("[a] Toggle auto-sync  [E] Change editor  [e] Edit config file"))
	b.WriteString("\n")

	return b.String()
}
