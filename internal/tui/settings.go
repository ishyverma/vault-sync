package tui

import (
	"fmt"
	"strings"
)

func (m model) settingsView() string {
	var b strings.Builder

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

	b.WriteString("Sync\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Auto-sync:  %v\n", m.config.Sync.AutoSync))
	b.WriteString(fmt.Sprintf("  Interval:   %ds\n", m.config.Sync.SyncInterval))
	b.WriteString(fmt.Sprintf("  Strategy:   %s\n", m.config.Sync.ConflictStrategy))

	return b.String()
}
