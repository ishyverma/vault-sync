# VaultSync

A terminal-first, Vim-powered note-taking app that syncs with Notion & Obsidian in real time.

Write in Vim. Save. It syncs — silently, instantly, everywhere.

```
vim my-note.md   →   :w   →   ✓ Synced to Notion + Obsidian
```

## Features

- **Local-first:** Everything works offline. Sync is additive, not required.
- **Vim-native:** No new editor. No new habits. Just your existing Vim workflow.
- **Zero friction:** The sync layer is invisible. You never think about it.
- **Portable:** Single binary. Drop it on any machine and it works.
- **Pluggable:** Notion and Obsidian today. Git, Linear, Logseq tomorrow.

## Quick Start

```bash
# Install
go install github.com/ishyverma/vault-sync/cmd/vault@latest

# Initialize
vault init

# Create your first note
vault new my-first-note

# Open the TUI dashboard
vault
```

## Commands

```
vault init          Interactive setup wizard
vault new [name]    Create + open a new note
vault open [name]   Fuzzy-find and open a note
vault list          List all notes
vault delete <name> Delete a note
vault search <q>    Full-text search across notes
vault daily         Open/create today's daily note
vault sync          Sync with connected backends
vault               Launch TUI dashboard
```

## License

MIT — see [LICENSE](LICENSE) for details.

Core features (local + Obsidian sync) are open source. Pro features (Notion sync) are source-available.
