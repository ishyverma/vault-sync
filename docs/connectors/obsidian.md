# Obsidian Connector

The Obsidian connector allows VaultSync to sync notes with an Obsidian vault.

## How It Works

Notes are copied from VaultSync's local storage into the configured Obsidian vault
folder. The connector uses file system operations (temp file + rename for atomic writes).

## Setup

1. Run: `vault connect obsidian --path ~/Documents/Obsidian/MyVault`
2. Optionally set a subfolder: `vault connect obsidian --path ... --subfolder "Notes"`

## Features

- Atomic file writes (no corruption on crash)
- Configurable subfolder within the vault
- Optional WikiLink support
- File watcher for detecting changes made inside Obsidian

## File Watcher

When the daemon is running (`vaultd start`), the Obsidian connector watches the
vault folder for changes. When a note is modified inside Obsidian, it's pulled
back into VaultSync's local storage and synced to other backends.
