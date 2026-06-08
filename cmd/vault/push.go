package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [filename]",
	Short: "Push a note to connected backends",
	Long: `Reads the note file, computes its hash, and syncs it to every
connected backend (Obsidian, Notion, etc.).

This command is called automatically by the Vim autocmd on save.
It can also be called manually:
  vault push my-note.md

Use --to to target a specific backend and --dry-run to preview.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		// If the path is absolute, resolve it relative to the notes directory.
		// This allows Vim/Neovim plugins (which pass expand('%:p')) to work.
		if filepath.IsAbs(filename) {
			mgr, err := newManager()
			if err == nil {
				notesDir := mgr.NotesDir()
				if rel, err := filepath.Rel(notesDir, filename); err == nil && !strings.HasPrefix(rel, "..") {
					filename = rel
				}
			}
		}

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

		to, _ := cmd.Flags().GetString("to")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		engine, err := newSyncEngine()
		if err != nil {
			return fmt.Errorf("sync engine: %w", err)
		}

		if to != "" {
			connectors := engine.Connectors()
			if _, ok := connectors[to]; !ok {
				return fmt.Errorf("backend not connected: %s", to)
			}
			if dryRun {
				fmt.Printf("[dry-run] Would push %s → %s\n", note.Filename, to)
				return nil
			}
		}

		if dryRun {
			fmt.Printf("[dry-run] Would push %s → all connected backends\n", note.Filename)
			return nil
		}

		if err := engine.PushNote(note.ID); err != nil {
			return fmt.Errorf("push note: %w", err)
		}

		label := ""
		if to != "" {
			label = fmt.Sprintf(" → %s", to)
		}
		fmt.Printf("✓ Pushed %s%s\n", note.Filename, label)
		return nil
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync all notes to connected backends",
	Long: `Pushes all local notes to every connected backend (Obsidian, Notion, etc.).

Notes that are already synced (matching content hash) are skipped automatically.
Use --force to re-push everything. Use --pull to also pull remote changes.
Use --flush-queue to process queued sync jobs (offline retries).
Use --to to target a specific backend and --dry-run to preview.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		doPull, _ := cmd.Flags().GetBool("pull")
		flushQueue, _ := cmd.Flags().GetBool("flush-queue")
		to, _ := cmd.Flags().GetString("to")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		engine, err := newSyncEngine()
		if err != nil {
			return fmt.Errorf("sync engine: %w", err)
		}

		if to != "" {
			connectors := engine.Connectors()
			if _, ok := connectors[to]; !ok {
				return fmt.Errorf("backend not connected: %s", to)
			}
		}

		if dryRun {
			target := "all connected backends"
			if to != "" {
				target = to
			}
			fmt.Printf("[dry-run] Would sync notes to %s\n", target)
			return nil
		}

		if flushQueue {
			n, err := engine.ProcessQueue()
			if err != nil {
				return fmt.Errorf("flush queue: %w", err)
			}
			if n > 0 {
				fmt.Printf("✓ Processed %d queued job(s)\n", n)
			}
		}

		if force {
			if err := engine.ExecutePreSyncHook(); err != nil {
				log.Printf("preSync hook: %v", err)
			}
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
					fmt.Fprintf(cmd.ErrOrStderr(), "⚠  %s: %v\n", n.Filename, err)
				} else {
					fmt.Printf("✓ %s\n", n.Filename)
				}
			}
			if err := engine.ExecutePostSyncHook(); err != nil {
				log.Printf("postSync hook: %v", err)
			}
		} else {
			if err := engine.SyncAll(); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "⚠  %v\n", err)
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
	pushCmd.Flags().String("to", "", "Target a specific backend (notion, obsidian, git)")
	pushCmd.Flags().Bool("dry-run", false, "Preview what would be pushed")
	syncCmd.Flags().BoolP("force", "f", false, "Re-push all notes regardless of sync state")
	syncCmd.Flags().Bool("pull", false, "Also pull remote changes after push")
	syncCmd.Flags().Bool("flush-queue", false, "Process queued sync jobs (offline retries)")
	syncCmd.Flags().String("to", "", "Target a specific backend (notion, obsidian, git)")
	syncCmd.Flags().Bool("dry-run", false, "Preview what would be synced")
}
