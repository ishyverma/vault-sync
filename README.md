<div align="center">
  <h1>VaultSync</h1>
  <p>
    <strong>Terminal-first, Vim-powered notes that sync everywhere</strong>
  </p>
  <p>
    <a href="https://github.com/ishyverma/vault-sync/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
    <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go" alt="Go"></a>
    <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-BubbleTea-ff69b4" alt="TUI"></a>
    <a href="https://sqlite.org"><img src="https://img.shields.io/badge/storage-SQLite-003B57" alt="Storage"></a>
  </p>
  <p>
    <code>vim my-note.md</code> &nbsp;→&nbsp; <code>:w</code> &nbsp;→&nbsp; <code>✓ Synced to Obsidian & Notion</code>
  </p>
  <br>
</div>

---

**VaultSync** is a local-first, terminal-native note-taking application. You write in Vim (or any editor), save, and it syncs — silently, instantly, to Obsidian **and** Notion. It features a full-screen terminal UI dashboard, full-text search, and pluggable sync backends.

---

## Features

### Core
- **Create, edit, list, delete, search** notes from the command line
- **YAML frontmatter** — title, date, tags auto-indexed in SQLite
- **Full-text search** via SQLite FTS4 with automatic LIKE fallback
- **Note templates** — blank, daily, meeting, project with `{{.Title}}` / `{{.Date}}` substitution
- **Daily notes** — `vault daily` opens today's dated note, creates it if missing

### Terminal UI (TUI)
- **6-tab dashboard** — Dashboard, Notes, Search, Sync, Settings, Conflicts
- **Note browser** — sortable, filterable table with rendered Glamour markdown preview
- **Live FTS search** — search-as-you-type with cursor navigation
- **Sync monitor** — per-backend sync state, pending queue, error display
- **Conflict resolver** — list conflicts, resolve with `l` (local) / `R` (remote) / `o` (edit)
- **Vim-style keybindings** — `j`/`k`, `1`-`6`, `Tab`, `/`, `?` help
- **Adaptive theming** — auto dark/light based on terminal background

### Obsidian Sync
- **Bidirectional sync** — push to and pull from any Obsidian vault
- **File-based connector** — atomic writes, path mapping, WikiLink support
- **Background daemon** (`vaultd`) — polls for changes at configurable intervals
- **Vim/Neovim plugins** — auto-sync on `:w` via `BufWritePost` autocmd
- **Sync state tracking** — per-note, per-backend status in SQLite

### Notion Sync
- **Bidirectional sync** — push to and pull from any Notion page
- **Full markdown↔blocks conversion** — headings, paragraphs, lists, code, tables, blockquotes, callouts, dividers, to-do, child pages
- **Inline formatting** — bold, italic, code, links converted to Notion annotations
- **Smart push** — creates new pages or updates existing ones with block replacement
- **Recursive pull** — traverses `has_children` blocks for complete content
- **YAML frontmatter round-trip** — title, tags, date preserved via Notion properties

### Storage
- **SQLite** — WAL mode, concurrent readers + writer with busy timeout
- **FTS4** — full-text search across title and content
- **Sync tables** — `sync_state`, `sync_queue`, `sync_history` for job tracking
- **Conflict detection** — hash comparison identifies divergent notes

---

## Quick Start

```bash
# Install
go install github.com/ishyverma/vault-sync/cmd/vault@latest

# Initialize your vault
vault init

# Create and open your first note
vault new my-first-note

# List all notes
vault list

# Search across notes
vault search "keyword"

# Launch the TUI dashboard
vault
```

### Connect Obsidian

```bash
vault connect obsidian --path ~/Documents/Obsidian/MyVault

# Push all notes to Obsidian
vault sync

# Start background daemon
vaultd start
```

### Connect Notion

```bash
vault connect notion --token ntn_xxxx --target-page-id <page-id>

# Push all notes to Notion
vault sync

# Pull latest from Notion
vault pull
```

---

## CLI Reference

| Command | Description |
|---------|-------------|
| `vault init` | Interactive setup — creates directories, config, welcome note |
| `vault new [name]` | Create a note from template and open in editor |
| `vault new [name] --template meeting` | Create using a specific template |
| `vault new [name] --no-open` | Create without opening editor |
| `vault open [name]` | Open a note by filename (fuzzy exact match) |
| `vault list` | Table view of all notes |
| `vault list --tag work` | Filter by tag |
| `vault delete [name]` | Delete a note (removes file, DB, sync state atomically) |
| `vault search [query]` | Full-text search across all notes |
| `vault daily` | Open or create today's daily note (`YYYY-MM-DD.md`) |
| `vault sync` | Sync all notes to all connected backends |
| `vault push [filename]` | Push a single note to all backends |
| `vault pull` | Pull all notes from all backends |
| `vault sync status` | Show per-note sync state |
| `vault connect obsidian --path [path]` | Configure Obsidian sync backend |
| `vault connect notion --token [key] --target-page-id [id]` | Configure Notion sync backend |
| `vault` / `vault ui` | Launch the terminal UI dashboard |
| `vaultd start` | Start background sync daemon |
| `vaultd stop` | Stop background daemon |
| `vaultd status` | Check daemon health |

---

## TUI Dashboard

The terminal UI provides six views accessible via number keys or Tab:

```
╭─ VaultSync ───────────────────────────────────────────────────╮
│                                                                │
│  📓 12 notes  |  ✍ 340 words today  |  28.0 KB  |  🔥 3 day  │
│                                                                │
│  Sync Status                                                   │
│  ✓ Synced:     12                                              │
│  Last sync: 2026-06-06 15:04                                   │
│                                                                │
│  Top Tags                                                      │
│    #daily (5)  #dev (3)  #work (2)                            │
│                                                                │
│  Recent Notes                                                  │
│    daily-2026-06-06.md — Daily Note (2026-06-06)               │
│    meeting-q1.md — Q1 Kickoff (2026-06-05)                     │
│                                                                │
│  ──────────────────────────────────────────────────────────    │
│  [n] New  [o] Open  [/] Search  [s] Sync  [?] Help  [q] Quit  │
├────────────────────────────────────────────────────────────────┤
│  📊 Dashboard │ 📝 Notes │ 🔍 Search │ 🔄 Sync │ ⚙ Settings   │
│  ↑/k up  ↓/j down  tab next  ? help  q quit                   │
╰────────────────────────────────────────────────────────────────╯
```

### Views

| View | Key | Description |
|------|-----|-------------|
| Dashboard | `1` | Stats, sync status, top tags, recent notes, quick actions |
| Notes | `2` | Sortable/filterable table with Glamour-rendered preview |
| Search | `3` | Live FTS search-as-you-type with result navigation |
| Sync | `4` | Backend status, sync state breakdown |
| Settings | `5` | Read-only config viewer |
| Conflicts | `6` | List and resolve sync conflicts |

### Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Tab` / `Shift+Tab` | Next / previous tab |
| `1`–`6` | Switch to view |
| `o` | Open note in editor |
| `s` | Sort (Notes) / Sync all (Dashboard) |
| `/` | Filter (Notes) / Search (other views) |
| `l` | Keep local (Conflicts) |
| `R` | Keep remote (Conflicts) |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       VaultSync System                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐    ┌───────────────────────────────────────┐  │
│   │  VIM / NVIM │    │       TUI (BubbleTea)                 │  │
│   │  BufWrite   │    │  Dashboard │ Browser │ Sync │ Config  │  │
│   └──────┬──────┘    └───────────────────┬───────────────────┘  │
│          │                               │                       │
│          ▼                               ▼                       │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                     CLI (Cobra)                          │  │
│   │  new │ open │ list │ delete │ search │ daily │ sync      │  │
│   └───────────────────────────┬──────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                     Core Engine                          │  │
│   │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│   │  │ Note Manager │  │ Sync Engine  │  │ Search (FTS4)  │ │  │
│   │  └──────────────┘  └──────┬───────┘  └────────────────┘ │  │
│   └───────────────────────────┼──────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                   Connectors                             │  │
│   │  ┌─────────────────────┐  ┌──────────────────────────┐   │  │
│   │  │  Obsidian Connector │  │  Notion Connector        │   │  │
│   │  └─────────────────────┘  └──────────────────────────┘   │  │
│   └──────────────────────────────────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                   Storage (SQLite)                       │  │
│   │  ~/.vault/notes/   ~/.vault/vault.db   ~/.vault/sync.db  │  │
│   │  ~/.config/vault/config.toml                              │  │
│   └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
:w in Vim
  → vault push <file> (called by BufWritePost autocmd)
    → Note Manager reads file, parses frontmatter
      → Update SQLite index (metadata + FTS)
             → Sync Engine computes checksum
               → If changed, enqueue SyncJob
                 → Worker pool processes queue
                   → Obsidian: atomic write to vault folder
                   → Notion: Markdown→Blocks conversion, API push
```

---

## Project Structure

```
vaultsync/
├── cmd/
│   ├── vault/              ← CLI entry point
│   │   ├── main.go         ← Root command, version
│   │   ├── init.go         ← vault init wizard
│   │   ├── new.go          ← vault new
│   │   ├── open.go         ← vault open
│   │   ├── list.go         ┢━ vault list
│   │   ├── delete.go       ┃  vault delete
│   │   ├── search.go       ┃  vault search
│   │   ├── daily.go        ┃  vault daily
│   │   ├── push.go         ┃  vault push / sync
│   │   ├── status.go       ┃  vault sync status
│   │   ├── connect.go      ┃  vault connect obsidian / notion
│   │   ├── ui.go           ┃  vault ui (TUI)
│   │   ├── app.go          ┃  Shared engine/manager setup
│   │   └── cmd_test.go     ┖  CLI integration tests
│   └── vaultd/
│       └── main.go         ← Background daemon entry point
├── internal/
│   ├── core/
│   │   ├── note.go         ← Note struct and constants
│   │   ├── manager.go      ← Note CRUD operations
│   │   ├── template.go     ← Template engine + 4 templates
│   │   └── frontmatter.go  ← YAML frontmatter parse/write
│   ├── sync/
│   │   ├── engine.go       ← Sync orchestrator
│   │   ├── conflict.go     ← Conflict detection & resolution
│   │   ├── queue.go        ← Job queue management
│   │   └── daemon.go       ← Background daemon
│   ├── connectors/
│   │   ├── connector.go    ← Connector interface
│   │   ├── obsidian/       ← Obsidian file-sync connector
│   │   ├── notion/         ← Notion API connector
│   │   └── git/            ← Git auto-commit connector (WIP)
│   ├── storage/
│   │   ├── store.go        ← SQLite connection, schema, CRUD
│   │   ├── note.go         ← Note model
│   │   ├── sync.go         ← Sync state/queue/history models
│   │   └── search.go       ← FTS search queries
│   ├── tui/
│   │   ├── model.go        ← BubbleTea model, update, view
│   │   ├── dashboard.go    ← Dashboard view
│   │   ├── browser.go      ← Note browser + preview
│   │   ├── search.go       ← Search results view
│   │   ├── synctab.go      ← Sync monitor view
│   │   ├── conflict.go     ← Conflict resolver view
│   │   ├── settings.go     ← Config viewer
│   │   ├── keys.go         ← Key bindings
│   │   ├── styles.go       ← Lipgloss styles
│   │   └── tui.go          ← Program entry point
│   └── config/
│       ├── config.go       ← Config struct, load, save
│       └── defaults.go     ← Default values
├── vim/
│   ├── vault.vim           ← Vimscript plugin
│   └── vault.lua           ← Neovim Lua plugin
├── Makefile                ← Build, test, lint, coverage
├── go.mod / go.sum
└── README.md
```

---

## Development

### Prerequisites

- Go 1.24+
- SQLite (included via `mattn/go-sqlite3`)

### Commands

```bash
make build        # Build vault binary to ./build/
make build-all    # Build both vault and vaultd
make test         # Run all tests with race detector + coverage
make lint         # Run golangci-lint
make cover        # Generate HTML coverage report
make install      # go install vault
make tidy         # go mod tidy && go mod verify
```

### Manual Testing

```bash
# Build and run
go build -o ./build/vault ./cmd/vault && ./build/vault

# Run all tests
go test -count=1 ./...

# Verify with race detector
go test -race -count=1 ./...
```

---

## Configuration

Config is stored at `~/.config/vault/config.toml` and generated by `vault init`:

```toml
[vault]
path = "~/.vault/notes"
editor = "nvim"

[sync]
sync_interval = 60
conflict_strategy = "ask"

[backends.obsidian]
enabled = true
vault_path = "~/Documents/Obsidian/MyVault"
subfolder = "VaultSync"
```

---

## Roadmap

| Phase | Status | Description |
|-------|--------|-------------|
| **Phase 0** | ✅ Complete | Core note management, SQLite, CLI commands, templates |
| **Phase 1** | ✅ Complete | Obsidian sync, daemon, Vim plugins, conflict detection |
| **Phase 2** | ✅ Complete | TUI Dashboard, note browser, search, conflict resolver |
| **Phase 3** | ✅ Complete | Notion sync via API (OAuth, markdown ↔ blocks conversion) |
| **Phase 4** | 🔜 Next | Advanced conflict resolution, offline queue, retry logic |
| **Phase 5** | ⏳ Planned | Version history, export, backlinks, graph view |
| **Phase 6** | ⏳ Planned | Distribution, Homebrew, install script, docs site |

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<div align="center">
  <sub>Built with ❤️ using <a href="https://github.com/charmbracelet/bubbletea">BubbleTea</a>, <a href="https://github.com/spf13/cobra">Cobra</a>, and <a href="https://sqlite.org">SQLite</a></sub>
</div>
