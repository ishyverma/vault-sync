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
Use --force to re-push everything.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

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

		fmt.Println("✓ Sync complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolP("force", "f", false, "Re-push all notes regardless of sync state")
}
