package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a note from your vault",
	Long: `Permanently deletes a note file and removes it from the index.

Examples:
  vault delete my-note
  vault delete old-meeting-notes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		if err := mgr.DeleteNote(name); err != nil {
			return fmt.Errorf("delete note: %w", err)
		}

		fmt.Printf("✓ Deleted %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
