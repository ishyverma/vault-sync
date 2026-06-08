package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var mvCmd = &cobra.Command{
	Use:   "mv [note-name] [folder]",
	Short: "Move a note to a folder",
	Long: `Moves a note into a folder/namespace. The folder is relative to
the vault notes directory and will be mirrored on remote backends.

Examples:
  vault mv my-note work
  vault mv my-note work/meetings`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		noteName := args[0]
		folder := args[1]

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := findNote(mgr, noteName)
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		oldPath := filepath.Join(mgr.NotesDir(), note.Filename)
		newFilename := note.Filename
		newFolder := strings.TrimSuffix(folder, "/")
		newPath := filepath.Join(mgr.NotesDir(), newFolder, newFilename)

		os.MkdirAll(filepath.Dir(newPath), 0o755)

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("move file: %w", err)
		}

		note.Path = filepath.Join(newFolder, newFilename)
		note.Folder = newFolder

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		if err := store.UpdateNote(note); err != nil {
			return fmt.Errorf("update store: %w", err)
		}

		fmt.Printf("✓ Moved %s → %s/\n", noteName, newFolder)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
}
