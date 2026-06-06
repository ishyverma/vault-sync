package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [filename]",
	Short: "Push a note to all connected backends",
	Long: `Reads the note file, computes its hash, and syncs it to every
connected backend (Obsidian, Notion, etc.).

This command is called automatically by the Vim autocmd on save.
It can also be called manually:
  vault push my-note.md

The filename is relative to the vault notes directory.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		if filepath.Ext(filename) == "" {
			filename += ".md"
		}

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, _, err := mgr.OpenNote(filename)
		if err != nil {
			return fmt.Errorf("note not found: %s", filename)
		}

		engine, err := newSyncEngine()
		if err != nil {
			return fmt.Errorf("sync engine: %w", err)
		}

		if err := engine.PushNote(note.ID); err != nil {
			return fmt.Errorf("push note: %w", err)
		}

		fmt.Printf("✓ Pushed %s\n", filename)
		return nil
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync all notes to connected backends",
	Long: `Pushes all local notes to every connected backend (Obsidian, Notion, etc.).

Notes that are already synced (matching content hash) are skipped automatically.
Use --force to re-push everything. Use --pull to also pull remote changes.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		doPull, _ := cmd.Flags().GetBool("pull")

		engine, err := newSyncEngine()
		if err != nil {
			return fmt.Errorf("sync engine: %w", err)
		}

		if force {
			mgr, err := newManager()
			if err != nil {
				return err
			}
			notes, err := mgr.ListNotes()
			if err != nil {
				return err
			}
			for _, n := range notes {
				if err := engine.PushNote(n.ID); err != nil {
					return fmt.Errorf("sync %s: %w", n.Filename, err)
				}
			}
		} else {
			if err := engine.SyncAll(); err != nil {
				return fmt.Errorf("sync all: %w", err)
			}
		}

		if doPull {
			if err := engine.PullAll(); err != nil {
				return fmt.Errorf("pull all: %w", err)
			}
		}

		fmt.Println("✓ Sync complete")
		return nil
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull remote changes from all connected backends",
	Long: `Fetches updates from every connected backend (Obsidian, Notion) and writes
them to the local notes directory.

Notes that haven't changed remotely are skipped. If both local and remote
have changed since the last sync, the conflict is flagged.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := newSyncEngine()
		if err != nil {
			return fmt.Errorf("sync engine: %w", err)
		}

		if err := engine.PullAll(); err != nil {
			return fmt.Errorf("pull all: %w", err)
		}

		fmt.Println("✓ Pull complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(pullCmd)
	syncCmd.Flags().BoolP("force", "f", false, "Re-push all notes regardless of sync state")
	syncCmd.Flags().Bool("pull", false, "Also pull remote changes after push")
}
