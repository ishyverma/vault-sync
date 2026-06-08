# Contributing to VaultSync

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

1. Install Go 1.22+
2. Clone the repo
3. Run `go build ./...` to verify everything compiles
4. Run `go test ./...` to run tests

## Project Structure

```
cmd/
  vault/        - CLI entry point and all commands
  vaultd/       - Background daemon
internal/
  core/         - Note management, templates, frontmatter
  sync/         - Sync engine, conflict resolution, queue
  connectors/   - Backend connectors (Notion, Obsidian, Git)
  storage/      - SQLite storage, FTS5 search
  tui/          - Terminal UI (BubbleTea)
  config/       - Config loading, credentials
vim/            - Vim/Neovim plugin source
scripts/        - Installation and setup scripts
```

## Coding Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use existing patterns for CLI commands (see `cmd/vault/*.go`)
- All exported functions need documentation comments
- Use testify for tests where appropriate

## Pull Request Process

1. Create a feature branch
2. Run `go build ./... && go vet ./... && go test ./...`
3. Submit a PR with a clear description of the changes

## Adding a New CLI Command

1. Create `cmd/vault/<name>.go`
2. Define `var <name>Cmd = &cobra.Command{...}`
3. Register in `init()` with `rootCmd.AddCommand(<name>Cmd)`
4. Use `newManager()` and `newStore()` helpers from `app.go`

## Adding a New Backend Connector

1. Create `internal/connectors/<name>/connector.go`
2. Implement the `connectors.Connector` interface
3. Add a compile-time check: `var _ connectors.Connector = (*Connector)(nil)`
