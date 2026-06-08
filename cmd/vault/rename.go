package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename [old-name] [new-name]",
	Short: "Rename a note",
	Long: `Renames a note file and updates its metadata.
The remote backend is also updated on the next sync.

Examples:
  vault rename old-name new-name
  vault rename meeting-q1 meeting-q1-2024`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		if !strings.HasSuffix(newName, ".md") {
			newName += ".md"
		}

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := findNote(mgr, oldName)
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		oldPath := filepath.Join(mgr.NotesDir(), note.Filename)
		newPath := filepath.Join(mgr.NotesDir(), newName)

		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("note already exists: %s", newName)
		}

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("rename file: %w", err)
		}

		note.Filename = newName
		note.Path = newName

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		if err := store.UpdateNote(note); err != nil {
			return fmt.Errorf("update store: %w", err)
		}

		fmt.Printf("✓ Renamed %s → %s\n", oldName, newName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
