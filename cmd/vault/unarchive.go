package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var unarchiveCmd = &cobra.Command{
	Use:   "unarchive [note-name]",
	Short: "Unarchive a note (show again in list)",
	Long: `Restores an archived note so it appears in the default list view
again. The note file and sync data are preserved.

Examples:
  vault unarchive my-note
  vault unarchive old-meeting-notes`,
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

		note.Archived = false

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		if err := store.UpdateNote(note); err != nil {
			return fmt.Errorf("unarchive note: %w", err)
		}

		fmt.Printf("✓ Unarchived %s\n", note.Filename)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unarchiveCmd)
}
