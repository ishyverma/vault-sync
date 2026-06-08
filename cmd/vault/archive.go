package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [note-name]",
	Short: "Archive a note (hide from list, keep in storage)",
	Long: `Archives a note so it no longer appears in the default list view.
The note file and sync data are preserved. Archived notes can be
viewed with vault list --archived.

Examples:
  vault archive my-note
  vault archive old-meeting-notes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := findNote(mgr, name)
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		note.Archived = true

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		if err := store.UpdateNote(note); err != nil {
			return fmt.Errorf("archive note: %w", err)
		}

		fmt.Printf("✓ Archived %s\n", note.Filename)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
}
