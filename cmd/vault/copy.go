package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy [note-name] [new-name]",
	Short: "Duplicate a note",
	Long: `Creates a copy of an existing note with a new name.
The new note inherits the content of the original.

Examples:
  vault copy my-note my-note-backup
  vault copy meeting-q1 meeting-q1-reviewed`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcName := args[0]
		dstName := args[1]

		if !strings.HasSuffix(dstName, ".md") {
			dstName += ".md"
		}

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		srcNote, err := findNote(mgr, srcName)
		if err != nil {
			return fmt.Errorf("find source note: %w", err)
		}

		srcPath := filepath.Join(mgr.NotesDir(), srcNote.Filename)
		dstPath := filepath.Join(mgr.NotesDir(), dstName)

		if _, err := os.Stat(dstPath); err == nil {
			return fmt.Errorf("note already exists: %s", dstName)
		}

		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read source note: %w", err)
		}

		if err := os.WriteFile(dstPath, content, 0o644); err != nil {
			return fmt.Errorf("write copy: %w", err)
		}

		dstNote, _, err := mgr.OpenNote(dstName)
		if err != nil {
			return fmt.Errorf("index copy: %w", err)
		}

		dstNote.Title = strings.TrimSuffix(dstName, ".md")
		dstNote.Tags = srcNote.Tags

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		if err := store.UpdateNote(dstNote); err != nil {
			return fmt.Errorf("update copy: %w", err)
		}

		fmt.Printf("✓ Copied %s → %s\n", srcNote.Filename, dstName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
