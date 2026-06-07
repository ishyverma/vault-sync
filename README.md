<div align="center">
  <h1>VaultSync</h1>
  <p>
    <strong>Terminal-first, Vim-powered notes that sync everywhere</strong>
  </p>
  <p>
    <a href="https://github.com/ishyverma/vault-sync/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
    <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go" alt="Go"></a>
    <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-BubbleTea-ff69b4" alt="TUI"></a>
    <a href="https://sqlite.org"><img src="https://img.shields.io/badge/storage-SQLite-003B57" alt="Storage"></a>
  </p>
  <p>
    <code>vim my-note.md</code> &nbsp;→&nbsp; <code>:w</code> &nbsp;→&nbsp; <code>✓ Synced to Obsidian & Notion</code>
  </p>
  <br>
</div>

---

**VaultSync** is a local-first, terminal-native note-taking application. Write in Vim (or any editor), save, and it syncs — silently, instantly — to Obsidian **and** Notion. It features a full-screen TUI dashboard, SQLite-backed full-text search, offline queue with exponential retry backoff, and pluggable sync backends.

---

## Features

### Core Note Management
- **Create, open, list, delete, search** notes from the command line
- **YAML frontmatter** — title, date, tags auto-indexed in SQLite with round-trip fidelity
- **Full-text search** via SQLite FTS4 with automatic LIKE fallback
- **Note templates** — blank, daily, meeting, project with `{{.Title}}` / `{{.Date}}` / `{{.Folder}}` substitution
- **Daily notes** — `vault daily` opens today's dated note, creates it if missing
- **Atomic file writes** — temp-file + rename for crash-safe persistence

### Obsidian Sync
- **Bidirectional sync** — push to and pull from any Obsidian vault
- **File-based connector** — atomic writes, path mapping, WikiLink support (`[[wikilinks]]`)
- **Background daemon** (`vaultd`) — file watcher + poller for remote changes
- **Vim/Neovim plugins** — auto-sync on `:w` via `BufWritePost` autocmd with `VaultSyncStatusline()` for statusbar integration
- **Sync state tracking** — per-note, per-backend status in SQLite (synced, pending, conflict, failed, local_only)

### Notion Sync
- **Bidirectional sync** — push to and pull from any Notion page
- **Full Markdown ↔ Blocks conversion** — headings, paragraphs, lists (bulleted/numbered/to-do), code blocks, tables, blockquotes, callouts, dividers, child pages, images, embeds, equations, columns, synced blocks
- **Inline formatting** — bold, italic, code, strikethrough, links, mentions converted to Notion rich-text annotations
- **Smart push** — creates new pages or updates existing ones with block replacement (diff-driven)
- **Recursive pull** — traverses `has_children` blocks for complete content
- **YAML frontmatter round-trip** — title, tags, date preserved via Notion properties
- **Database sync mode** — push notes as database rows with schema-aware property mapping (title, multi_select tags, date, rich_text, select, status)
- **Workspace-level page ID tracking** — maps local filenames ↔ Notion page IDs in SQLite sync state
- **Rate limit handling** — respects Notion's 3 req/sec limit via queued retries

### Conflict Resolution
- **Automatic detection** — per-backend canonical hash comparison (embed-aware for Notion, raw hash for filesystem)
- **CLI resolver** — `vault conflicts` lists all conflicts, `--resolve local|remote` auto-resolves
- **TUI diff view** — `d` or `Enter` shows side-by-side unified diff via go-difflib
- **Resolution strategies** — keep local, keep remote, or open in editor to manually merge
- **Vim integration** — `o` in TUI opens the conflicted note in Vim for manual editing

### Offline Queue & Retry
- **Automatic enqueue** — connectivity errors (DNS, connection refused, timeout, TLS handshake) enqueue failed jobs
- **Exponential backoff** — `min(2^(attempt-1), 30)` second delays between retries
- **Configurable retry limit** — `queue_retry_limit = 5` in config, jobs exceeding limit are marked as failed
- **Auto-flush** — `SyncAll` processes the queue at start and end; `--flush-queue` flag on `sync` for manual flush
- **SQLite-backed queue** — persistent across restarts, items sorted by `queued_at` with future-date backoff support

### Terminal UI (TUI)
- **6-tab dashboard** — Dashboard, Notes, Search, Sync, Settings, Conflicts
- **Dashboard** — note count, word count, sync status, top tags, recent notes, quick actions
- **Note browser** — sortable/filterable table with Glamour-rendered markdown preview in a split pane
- **Live FTS search** — search-as-you-type with cursor navigation through results
- **Sync monitor** — per-backend connector status, queue length, recent sync history log
- **Conflict resolver** — list with `l` (keep local) / `R` (keep remote) / `d` (view diff) / `o` (open in editor)
- **Settings viewer** — read-only display of current configuration
- **Vim-style keybindings** — `j`/`k`, `1`-`6`, `Tab`, `/`, `?` help
- **Error notifications** — toast-style 3-second info banner on sync completion/failure
- **Lipgloss theming** — adaptive dark/light with columnar layouts

### Storage
- **SQLite** — WAL journal mode, concurrent readers + writer with 5s busy timeout, pure Go driver (no CGO required)
- **FTS4** — full-text search across title and content with unicode61 tokenizer (LIKE fallback when FTS not available)
- **Schema** — `notes` (metadata, tags, FTS), `sync_state` (per-backend tracking), `sync_queue` (offline retries), `sync_history` (audit log), `versions` (snapshots)

---

## Quick Start

### macOS — Homebrew

```bash
brew install ishyverma/tap/vault-sync
```

### Linux — APT / RPM / APK

Download the `.deb`, `.rpm`, or `.apk` from the [latest release](https://github.com/ishyverma/vault-sync/releases/latest).

### Any platform — one-liner

```bash
curl -fsSL https://vaultsync.dev/install | sh
```

### Go install

```bash
go install github.com/ishyverma/vault-sync/cmd/vault@latest
```

---

```bash
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
# Push to a parent page (default mode)
vault connect notion --token ntn_xxxx --target-page-id <page-id>
vault sync

# Push as database rows (optional, set database_id in config)
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
| `vault sync --force` | Re-push all notes regardless of sync state |
| `vault sync --flush-queue` | Process queued sync jobs (offline retries) |
| `vault push [filename]` | Push a single note to all backends |
| `vault pull` | Pull all notes from all backends |
| `vault sync status` | Show per-note sync state |
| `vault conflicts` | List all sync conflicts |
| `vault conflicts --resolve local` | Auto-resolve all conflicts keeping local version |
| `vault conflicts --resolve remote` | Auto-resolve all conflicts keeping remote version |
| `vault connect obsidian --path [path]` | Configure Obsidian sync backend |
| `vault connect notion --token [key] --target-page-id [id]` | Configure Notion sync backend |
| `vault` / `vault ui` | Launch the TUI dashboard |
| `vaultd start` | Start background sync daemon |
| `vaultd stop` | Stop background daemon |
| `vaultd status` | Check daemon health |

---

## TUI Dashboard

The TUI provides six views accessible via **number keys `1`–`6`** or **Tab**:

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

### Conflict Resolver Diff View

Press **`d`** or **`Enter`** on a conflicted note to view the unified diff:

```
╭─ Conflict Diff ───────────────────────────────────────────────╮
│  --- Local                                                     │
│  +++ Notion                                                    │
│  @@ -1,5 +1,6 @@                                              │
│   # Daily Note - Jun 07                                        │
│                                                                │
│   ## Morning                                                   │
│  - Reviewed PRs                                                │
│  + Standup call at 10am                                        │
│  + Auth bug assigned to @me                                    │
│                                                                │
│   ## Tasks                                                     │
│  [esc] Back to conflict list  [j/k] Scroll                     │
╰────────────────────────────────────────────────────────────────╯
```

### Views

| View | Key | Description |
|------|-----|-------------|
| Dashboard | `1` | Stats, sync status, top tags, recent notes, quick actions |
| Notes | `2` | Sortable/filterable table with Glamour-rendered preview |
| Search | `3` | Live FTS search-as-you-type with result navigation |
| Sync | `4` | Connector status, queue length, recent sync history |
| Settings | `5` | Read-only config viewer |
| Conflicts | `6` | List conflicts, view diff, resolve with local/remote/edit |

### Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Tab` / `Shift+Tab` | Next / previous tab |
| `1` – `6` | Switch to view |
| `o` | Open note in editor |
| `s` | Sort (Notes) / Sync all (Dashboard) |
| `/` | Filter (Notes) / Search (other views) |
| `l` / `R` | Keep local / remote (Conflicts) |
| `d` / `Enter` | View conflict diff (Conflicts) |
| `r` | Refresh (Sync / Conflicts) |
| `f` | Force sync all (Sync) |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

---

## Vim Integration

### Auto-sync on save

**Vimscript** (`source ~/.config/vault/vault.vim`):
```vim
augroup VaultSync
  autocmd!
  autocmd BufWritePost ~/.vault/notes/*.md
        \ silent! call s:push(expand("<afile>"))
augroup END
```

**Neovim Lua** (`require('vault')` or via lazy.nvim):
```lua
vim.api.nvim_create_autocmd('BufWritePost', {
  pattern = vim.fn.expand('~/.vault/notes/*.md'),
  callback = function(ev) require('vault').push(ev.file) end,
})
```

### Statusline Integration

Show sync status in your Vim statusline:

```vim
" Add to your .vimrc:
set statusline+=%{VaultSyncStatusline()}
" Returns: ' ✓' / ' ⚠' / ' ✗' / ' ⟳' / ''
```

```lua
-- Neovim (lualine example):
local vault = require('vault')
table.insert(sections.lualine_x, { vault.statusline })
```

Commands: `:VaultSyncPush`, `:VaultSyncStatus`

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
│   │  push │ pull │ connect │ conflicts │d start|stop|status  │  │
│   └───────────────────────────┬──────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                     Core Engine                          │  │
│   │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐  │  │
│   │  │ Note Manager │  │ Sync Engine  │  │ Search (FTS4)  │  │  │
│   │  │              │  │              │  │                │  │  │
│   │  │ create/edit  │  │ queue mgmt   │  │ SQLite FTS     │  │  │
│   │  │ delete/move  │  │ conflict res │  │ fuzzy matching │  │  │
│   │  │ template eng │  │ retry logic  │  │                │  │  │
│   │  └──────────────┘  └──────┬───────┘  └────────────────┘  │  │
│   └───────────────────────────┼──────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                   Connectors                             │  │
│   │  ┌─────────────────────┐  ┌──────────────────────────┐   │  │
│   │  │  Obsidian Connector │  │  Notion Connector        │   │  │
│   │  │                     │  │                          │   │  │
│   │  │  atomic file copy   │  │  OAuth + REST API        │   │  │
│   │  │  path mapping       │  │  md↔blocks conversion    │   │  │
│   │  │  file watcher       │  │  database sync mode      │   │  │
│   │  │  WikiLinks          │  │  page ID mapping         │   │  │
│   │  └─────────────────────┘  └──────────────────────────┘   │  │
│   └──────────────────────────────────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                   Storage (SQLite)                       │  │
│   │  files: ~/.vault/notes/*.md                              │  │
│   │  db:    ~/.vault/vault.db (notes, FTS, sync_state,      │  │
│   │                           sync_queue, sync_history)      │  │
│   │  config: ~/.config/vault/config.toml                     │  │
│   └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
:w in Vim
  → vault push <file> (BufWritePost autocmd)
    → Note Manager reads file, parses frontmatter
      → Update SQLite index (metadata + FTS)
        → Sync Engine computes canonical hash
          → If changed from last-synced hash:
            → For each backend:
              → Check remote for conflict (hash comparison)
              → Push note (create or update)
              → On connectivity error → enqueue to offline queue
                → Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s
                → Retry limit of 5, then mark as failed
              → Record sync state + history
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
│   │   ├── list.go         ← vault list
│   │   ├── delete.go       ← vault delete
│   │   ├── search.go       ← vault search
│   │   ├── daily.go        ← vault daily
│   │   ├── push.go         ← vault push / sync
│   │   ├── pull.go         ← vault pull
│   │   ├── conflicts.go    ← vault conflicts
│   │   ├── status.go       ← vault sync status
│   │   ├── connect.go      ← vault connect obsidian / notion
│   │   ├── ui.go           ← vault ui (TUI)
│   │   ├── app.go          ← Shared engine/manager setup
│   │   └── cmd_test.go     ← CLI integration tests
│   └── vaultd/
│       └── main.go         ← Background daemon entry point
├── internal/
│   ├── core/
│   │   ├── note.go         ← Note struct and constants
│   │   ├── manager.go      ← Note CRUD operations, GetNote
│   │   ├── template.go     ← Template engine + 4 templates
│   │   └── frontmatter.go  ← YAML frontmatter parse/write (deterministic struct-based)
│   ├── sync/
│   │   ├── engine.go       ← Sync orchestrator (push, pull, conflict detection,
│   │   │                      canonical hashing, offline queue, ProcessQueue)
│   │   ├── conflict.go     ← Conflict detection & resolution (ResolveConflict)
│   │   ├── daemon.go       ← Background daemon (file watcher + poller)
│   │   └── engine_test.go  ← Sync tests (push, pull, conflicts, status, failures)
│   ├── connectors/
│   │   ├── connector.go    ← Connector interface (Connect, Push, Pull, Delete, Status)
│   │   ├── obsidian/       ← Obsidian file-sync connector
│   │   │   ├── connector.go
│   │   │   ├── watcher.go
│   │   │   └── wikilinks.go
│   │   └── notion/         ← Notion API connector
│   │       ├── client.go   ← API client (GetDatabase, QueryDatabase, pages, blocks)
│   │       ├── convert.go  ← Markdown ↔ Notion blocks (goldmark AST)
│   │       ├── connector.go← Push, Pull, pushToDatabase, buildDBProperties
│   │       ├── auth.go     ← OAuth flow
│   │       ├── types.go    ← Notion API types (Database, PropertyConfig, Page, Block)
│   │       └── mapper.go   ← note_id ↔ page_id mapping
│   ├── storage/
│   │   ├── store.go        ← SQLite connection, schema migration, CRUD
│   │   ├── note.go         ← Note model
│   │   ├── sync.go         ← SyncState, SyncQueueItem, SyncHistoryEntry
│   │   ├── search.go       ← FTS search queries
│   │   ├── errors.go       ← Sentry errors (ErrNoteNotFound, ErrSyncJobNotFound, etc.)
│   │   ├── sync_test.go    ← Sync state/queue/history tests
│   │   └── cmd_test.go     ← Storage integration tests
│   ├── tui/
│   │   ├── model.go        ← BubbleTea model, update loop, view renderer
│   │   ├── dashboard.go    ← Dashboard view (stats, recent, sync status)
│   │   ├── browser.go      ← Note browser + Glamour preview
│   │   ├── search.go       ← Search results view
│   │   ├── synctab.go      ← Sync monitor (connector status, queue, history)
│   │   ├── conflict.go     ← Conflict resolver (list, diff, resolve)
│   │   ├── settings.go     ← Config viewer
│   │   ├── keys.go         ← Key bindings
│   │   ├── styles.go       ← Lipgloss styles (adaptive dark/light)
│   │   └── tui.go          ← Program entry point
│   └── config/
│       ├── config.go       ← Config struct, load, save
│       └── defaults.go     ← Default values (retry_limit, sync_interval, etc.)
├── vim/
│   ├── vault.vim           ← Vimscript plugin (auto-push, statusline function)
│   └── vault.lua           ← Neovim Lua plugin (auto-push, statusline function)
├── Makefile                ← Build, test, lint, coverage, install
├── go.mod / go.sum
└── README.md
```

---

## Configuration

Full config at `~/.config/vault/config.toml` (generated by `vault init`):

```toml
[vault]
path = "~/.vault/notes"
editor = "nvim"
template_dir = "~/.vault/templates"
default_template = "blank"
auto_daily = true

[sync]
sync_interval = 60
conflict_strategy = "ask"
queue_retry_limit = 5

[backends.notion]
enabled = true
token = ""
target_page_id = ""
database_id = ""
sync_direction = "both"

[backends.obsidian]
enabled = true
vault_path = "~/Documents/Obsidian/MyVault"
subfolder = "VaultSync"
wikilinks = true
sync_direction = "both"

[search]
fuzzy = true
max_results = 50

[tui]
theme = "dark"
date_format = "2006-01-02"
```

---

## Development

### Prerequisites

- Go 1.24+
- Pure Go SQLite via `modernc.org/sqlite` (no CGO required)

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

## Roadmap

| Phase | Status | Description |
|-------|--------|-------------|
| **Phase 0** | ✅ Complete | Core note management, SQLite, CLI commands, templates |
| **Phase 1** | ✅ Complete | Obsidian sync, daemon, Vim plugins |
| **Phase 2** | ✅ Complete | TUI Dashboard, note browser, search, settings |
| **Phase 3** | ✅ Complete | Notion sync (OAuth, markdown↔blocks, database mode) |
| **Phase 4** | ✅ Complete | Conflict resolution, offline queue, exponential backoff, sync monitor, statusline |
| **Phase 5** | ✅ Complete | Version history, import/export, backlinks, graph view, shell hooks |
| **Phase 6** | ✅ Complete | Distribution: GoReleaser, Homebrew, AUR, APT/RPM/APK packages, install script, Neovim plugin |

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<div align="center">
  <sub>Built with ❤️ using <a href="https://github.com/charmbracelet/bubbletea">BubbleTea</a>, <a href="https://github.com/spf13/cobra">Cobra</a>, <a href="https://github.com/yuin/goldmark">Goldmark</a>, and <a href="https://sqlite.org">SQLite</a></sub>
</div>
