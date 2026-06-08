<div align="center">
  <h1>VaultSync</h1>
  <p>
    <strong>Terminal-first, Vim-powered notes that sync everywhere</strong>
  </p>
  <p>
    <code>vim my-note.md</code> &nbsp;→&nbsp; <code>:w</code> &nbsp;→&nbsp; <code>✓ Synced to Obsidian & Notion</code>
  </p>
  <p>
    <a href="https://github.com/ishyverma/vault-sync/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
    <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go" alt="Go"></a>
  </p>
</div>

---

**VaultSync** is a local-first, terminal-native note-taking application. Write in Vim, save, and it syncs — silently, instantly — to Obsidian **and** Notion. It runs as a single binary with zero dependencies.

---

## Quick Start

### macOS — Homebrew

```bash
brew install ishyverma/tap/vault-sync
```

### Any platform — Go install

```bash
go install github.com/ishyverma/vault-sync/cmd/vault@latest
go install github.com/ishyverma/vault-sync/cmd/vaultd@latest
```

### Linux — APT / RPM / APK

Download from the [latest release](https://github.com/ishyverma/vault-sync/releases/latest).

### Initialize your vault

```bash
vault init

# Create and open your first note
vault new my-first-note

# Launch the TUI dashboard
vault
```

---

## Features

### Core Note Management
- **Create, open, list, delete, search** notes from the command line
- **YAML frontmatter** — title, date, tags auto-indexed in SQLite with round-trip fidelity
- **Full-text search** via SQLite FTS5 with automatic LIKE fallback when FTS is unavailable
- **Note templates** — blank, daily, meeting, project with `{{.Title}}` / `{{.Date}}` / `{{.Folder}}` substitution
- **Daily notes** — `vault daily` opens today's dated note, creates it if missing
- **Word count** — tracked per-note in the database, displayed in `vault list`
- **Tags** — YAML frontmatter tags searchable and filterable via `vault list --tag` and `vault search --tag`
- **Pinned notes** — toggle pin in the TUI note browser with `p`, filter with `P`
- **Atomic file writes** — temp-file + rename for crash-safe persistence
- **Encrypted credentials** — Notion token encrypted with AES-256-GCM, stored separately from config

### Terminal UI (TUI)
- **6-tab dashboard** — Dashboard, Notes, Search, Sync, Settings, Conflicts
- **Dashboard** — note count, word count, sync status, top tags, recent notes, writing streak, quick actions
- **Note browser** — sortable/filterable table with Glamour-rendered markdown preview in a split pane, pin toggle
- **Live FTS search** — search-as-you-type with cursor navigation through results
- **Sync monitor** — per-backend connector status, queue length, recent sync history log
- **Conflict resolver** — list with `l` (keep local) / `R` (keep remote) / `d` (view diff) / `o` (open in editor)
- **Settings viewer** — read-only display of current configuration; press `e` to edit in your configured editor
- **Vim-style keybindings** — `j`/`k`, `1`-`6`, `Tab`, `/`, `?` help, `s` sort, `o` open
- **Error notifications** — toast-style 3-second info banner on sync completion/failure
- **Lipgloss theming** — adaptive dark/light with columnar layouts

### Obsidian Sync
- **Bidirectional sync** — push to and pull from any Obsidian vault
- **File-based connector** — atomic writes with path mapping to configured subfolder
- **Folder mapping** — local note folder structure mirrored in Obsidian
- **Background daemon** (`vaultd`) — file watcher + poller for remote changes
- **Sync state tracking** — per-note, per-backend status in SQLite (synced, pending, conflict, failed, local_only)

### Notion Sync
- **Bidirectional sync** — push to and pull from any Notion page
- **Full Markdown ↔ Blocks conversion** — headings, paragraphs, lists (bulleted/numbered/to-do), code blocks, tables, blockquotes, callouts, dividers, child pages, images, embeds, equations
- **Inline formatting** — bold, italic, code, strikethrough, links converted to Notion rich-text annotations
- **Smart push** — creates new pages or updates existing ones
- **Recursive pull** — traverses `has_children` blocks for complete content
- **YAML frontmatter round-trip** — title, tags, date preserved via Notion properties
- **Database sync mode** — push notes as database rows with schema-aware property mapping (title, multi_select tags, date, rich_text)
- **Page ID tracking** — maps local filenames ↔ Notion page IDs in SQLite sync state
- **Rate limit handling** — respects Notion's 3 req/sec limit via queued retries with exponential backoff
- **Encrypted token storage** — Notion API token encrypted at rest with AES-256-GCM

### Offline Queue & Retry
- **Automatic enqueue** — connectivity errors enqueue failed jobs automatically
- **Exponential backoff** — `min(2^(attempt-1), 30)` second delays between retries
- **Configurable retry limit** — `queue_retry_limit` in config
- **SQLite-backed queue** — persistent across restarts with future-date backoff support
- **Manual flush** — `vault sync --flush-queue` processes queued jobs

### Version History
- **Auto-versioning** — snapshot saved before every sync push, pull, and conflict resolution
- **`vault history <note>`** — list all saved versions with timestamps and triggers
- **`vault restore <note> --version <n>`** — restore a note to a previous version, creates a new version for the restore event
- **Git integration** — optional Git connector with auto-commit support

### Search & Discovery
- **SQLite FTS5** — full-text search across title and content with `unicode61` tokenizer
- **Tag search** — `vault search --tag <tag>` and `vault list --tag <tag>`
- **Backlinks** — `vault backlinks <note>` finds all notes linking to this one via `[[WikiLink]]`
- **Graph view** — `vault graph` displays WikiLink connections between notes
- **JSON output** — `--json` on list, search, status, conflicts, backlinks, graph, and history

### Export & Import
- **HTML export** — `vault export html <note>` renders markdown to styled HTML via goldmark
- **PDF export** — `vault export pdf <note>` converts via pandoc
- **Notion import** — `vault import notion` bulk-imports existing Notion pages

### Conflict Resolution
- **Automatic detection** — per-backend canonical hash comparison
- **CLI resolver** — `vault conflicts` lists all conflicts, `--resolve local|remote` auto-resolves
- **TUI diff view** — unified diff with keyboard-driven resolution
- **Resolution strategies** — keep local, keep remote, open in editor, or last-write-wins
- **Configurable default** — `conflict_strategy` in config: `ask` (default), `local_wins`, `remote_wins`, `last_write_wins`

### Vim Integration
- **Auto-sync on save** — `BufWritePost` autocmd pushes notes to all backends
- **`:VaultSyncPush`** — manually push the current buffer
- **`:VaultSyncStatus`** — show sync status for all notes
- **Statusline function** — `VaultSyncStatusline()` shows sync status per-note (synced/conflict/failed/pending)
- **lazy.nvim compatible** — install as a Neovim plugin with full setup API
- **vim-plug compatible** — standard Vim plugin structure

### Shell Hooks
- **`pre_sync`** — shell command executed before every sync cycle
- **`post_sync`** — shell command executed after every sync cycle
- **`on_conflict`** — shell command executed when a conflict is detected

---

## Vim Plugin Installation

### Option 1: Auto-install via vault

```bash
# Install the vault binary first, then run:
vault plugin install

# Or specify the target:
vault plugin install --vim      # Install for Vim
vault plugin install --neovim   # Install for Neovim
```

This copies the plugin files to Vim's `packpath` directory. Restart your editor and `:VaultSyncPush` will work immediately.

### Option 2: lazy.nvim (Neovim)

```lua
{
  'ishyverma/vault-sync',
  dir = 'vim',
  config = function()
    require('vault').setup()
  end,
}
```

### Option 3: vim-plug (Vim)

```vim
Plug 'ishyverma/vault-sync', { 'dir': 'vim' }
```

---

## CLI Reference

| Command | Description |
|---------|-------------|
| Command | Description |
|---------|-------------|
| `vault init` | Interactive setup — creates directories, config, welcome note |
| `vault status` | Show connection status of all backends |
| `vault status --json` | JSON output |
| `vault connect notion --token [key]` | Configure Notion sync backend |
| `vault connect obsidian --path [path]` | Configure Obsidian sync backend |
| `vault disconnect [backend]` | Remove a backend connection |
| `vault new [name]` | Create a note from template and open in editor |
| `vault new --template meeting` | Create using a specific template |
| `vault new --no-open` | Create without opening editor |
| `vault open [name]` | Open a note by filename in your configured editor |
| `vault list` | Table view of all notes |
| `vault list --tag work` | Filter by tag |
| `vault list --archived` | Show archived notes |
| `vault list --json` | JSON output for scripting |
| `vault delete [name]` | Delete a note (removes file, DB, sync state atomically) |
| `vault rename [old] [new]` | Rename a note |
| `vault mv [note] [folder]` | Move a note to a folder |
| `vault copy [note] [new-name]` | Duplicate a note |
| `vault archive [name]` | Archive a note (hide from list) |
| `vault unarchive [name]` | Unarchive a note (show in list) |
| `vault daily` | Open or create today's daily note |
| `vault quick` | Open a quick scratch buffer |
| `vault search [query]` | Full-text search across all notes |
| `vault search --tag work` | Filter search results by tag |
| `vault search --after 2024-01-01` | Search notes created after date |
| `vault search --before 2024-06-01` | Search notes created before date |
| `vault search --regex "pattern"` | Regex search on title/filename |
| `vault search --json` | JSON output |
| `vault sync` | Sync all notes to all connected backends |
| `vault sync --force` | Re-push all notes regardless of sync state |
| `vault sync --pull` | Also pull remote changes after push |
| `vault sync --to notion` | Sync to only one backend |
| `vault sync --dry-run` | Preview changes without applying |
| `vault sync --flush-queue` | Process queued sync jobs (offline retries) |
| `vault sync status` | Show per-note sync state per backend |
| `vault sync status --json` | JSON output |
| `vault push [filename]` | Push a single note to all backends (called by Vim autocmd) |
| `vault pull` | Pull all notes from all backends |
| `vault conflicts` | List all sync conflicts |
| `vault conflicts --resolve local` | Auto-resolve all conflicts keeping local version |
| `vault conflicts --resolve remote` | Auto-resolve all conflicts keeping remote version |
| `vault conflicts --json` | JSON output |
| `vault export html [note]` | Export a note as styled HTML |
| `vault export pdf [note]` | Export a note as PDF (requires pandoc) |
| `vault export all html` | Export all notes as HTML |
| `vault import notion` | Bulk-import existing Notion pages |
| `vault import obsidian --path [path]` | Bulk-import from an Obsidian vault |
| `vault history [note]` | List version history for a note |
| `vault history [note] --json` | JSON output |
| `vault restore [note] --version [n]` | Restore a note to a previous version |
| `vault diff [note]` | Show diff between last two versions |
| `vault diff [note] --v1 1 --v2 3` | Diff specific versions |
| `vault backlinks [note]` | Find all notes linking to this one |
| `vault backlinks [note] --json` | JSON output |
| `vault recent` | Show recently modified notes |
| `vault graph` | Display WikiLink connection graph |
| `vault graph --json` | JSON output |
| `vault config` | Open config file in your configured editor |
| `vault plugin install` | Install the Vim/Neovim plugin |
| `vault plugin install --vim` | Install for Vim |
| `vault plugin install --neovim` | Install for Neovim |
| `vault` / `vault ui` / `vault tui` | Launch the TUI dashboard |
| `vaultd start` | Start background sync daemon |
| `vaultd start --interval 120` | Start with custom poll interval |
| `vaultd stop` | Stop background daemon |
| `vaultd status` | Check daemon health |
| `vaultd install` | Install as launchd/systemd service |

---

## TUI Dashboard

The TUI provides six views accessible via **number keys `1`–`6`** or **Tab**:

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
| `p` / `P` | Toggle pin / Show pinned only |
| `l` / `R` | Keep local / remote (Conflicts) |
| `d` / `Enter` | View conflict diff (Conflicts) |
| `r` | Refresh (Sync / Conflicts) |
| `f` | Force sync all (Sync) |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

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
auto_sync = true
sync_interval = 60
conflict_strategy = "ask"
queue_retry_limit = 5

[backends.notion]
enabled = true
token = ""
target_page_id = ""
database_id = ""

[backends.obsidian]
enabled = true
vault_path = "~/Documents/Obsidian/MyVault"
subfolder = "VaultSync"
wikilinks = true

[backends.git]
enabled = false
repo_path = "~/.vault"
auto_commit = false
commit_message = "vault: sync {filename}"
remote = ""

[hooks]
pre_sync = ""
post_sync = ""
on_conflict = ""

[tui]
theme = "dark"
date_format = "2006-01-02"

[search]
fuzzy = true
max_results = 50

[notifications]
sync_success = false
sync_failure = true
conflict_detected = true
```

Credentials (Notion token) are stored encrypted in `~/.config/vault/credentials.json`.

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
│   │  │ Note Manager │  │ Sync Engine  │  │ Search (FTS5)  │  │  │
│   │  │              │  │              │  │                │  │  │
│   │  │ create/edit  │  │ queue mgmt   │  │ SQLite FTS     │  │  │
│   │  │ delete/move  │  │ conflict res │  │ full-text      │  │  │
│   │  │ template eng │  │ retry logic  │  │ backlinks      │  │  │
│   │  └──────────────┘  └──────┬───────┘  └────────────────┘  │  │
│   └───────────────────────────┼──────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                   Connectors                             │  │
│   │  ┌─────────────────────┐  ┌──────────────────────────┐   │  │
│   │  │  Obsidian Connector │  │  Notion Connector        │   │  │
│   │  │                     │  │                          │   │  │
│   │  │  atomic file copy   │  │  REST API + rate limit   │   │  │
│   │  │  path mapping       │  │  md↔blocks conversion    │   │  │
│   │  │  file watcher       │  │  database sync mode      │   │  │
│   │  └─────────────────────┘  └──────────────────────────┘   │  │
│   │  ┌────────────────────────────────────────────────────┐   │  │
│   │  │  Git Connector (optional)                          │   │  │
│   │  │  auto-commit + remote push/pull                    │   │  │
│   │  └────────────────────────────────────────────────────┘   │  │
│   └──────────────────────────────────────────────────────────┘  │
│                               │                                  │
│                               ▼                                  │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │                   Storage (SQLite)                       │  │
│   │  files: ~/.vault/notes/*.md                              │  │
│   │  db:    ~/.vault/vault.db (notes, FTS, sync_state,      │  │
│   │                           sync_queue, sync_history,      │  │
│   │                           versions, tags)                │  │
│   │  config: ~/.config/vault/config.toml                     │  │
│   │  credentials: ~/.config/vault/credentials.json          │  │
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

## Development

### Prerequisites

- Go 1.24+
- No CGO required — pure Go SQLite via `modernc.org/sqlite`

### Commands

```bash
make build        # Build vault binary to ./build/
make build-all    # Build both vault and vaultd
make test         # Run all tests with race detector + coverage
make cover        # Generate HTML coverage report
make install      # go install vault and vaultd
make lint         # Run golangci-lint
make tidy         # go mod tidy && go mod verify
```

### Manual Testing

```bash
go build -o ./build/vault ./cmd/vault && ./build/vault
go test -count=1 -race ./...
```

### Project Structure

```
vaultsync/
├── cmd/
│   ├── vault/              ← CLI entry point
│   └── vaultd/             ← Daemon entry point
├── internal/
│   ├── core/               ← Note CRUD, templates, frontmatter
│   ├── sync/               ← Sync engine, daemon, queue processing
│   ├── connectors/
│   │   ├── notion/         ← Notion API client + Markdown↔Blocks
│   │   ├── obsidian/       ← Obsidian file-copy connector
│   │   └── git/            ← Optional git auto-commit connector
│   ├── storage/            ← SQLite schema, CRUD, FTS5, sync state
│   ├── tui/                ← BubbleTea TUI
│   └── config/             ← Config loading, defaults, credentials
├── vim/                    ← Vim/Neovim plugin files
├── lua/                    ← Lazy.nvim entry point
├── contrib/
│   ├── homebrew/           ← Homebrew formula
│   └── aur/               ← AUR package
├── scripts/                ← Installer script
├── embed.go                ← Embedded plugin files for `vault plugin install`
├── .goreleaser.yaml        ← Release pipeline
└── Makefile
```

---

## Storage

- **SQLite** — WAL journal mode, concurrent readers + writer with 5s busy timeout, pure Go driver (no CGO)
- **FTS5** — full-text search across title and content with `unicode61` tokenizer
- **Schema** — `notes` (metadata, tags, FTS), `sync_state` (per-backend tracking), `sync_queue` (offline retries), `sync_history` (audit log), `versions` (snapshots)

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<div align="center">
  <sub>Built with <a href="https://github.com/charmbracelet/bubbletea">BubbleTea</a>, <a href="https://github.com/spf13/cobra">Cobra</a>, <a href="https://github.com/yuin/goldmark">Goldmark</a>, and <a href="https://sqlite.org">SQLite</a></sub>
</div>
