package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [name]",
	Short: "Open a note in your editor",
	Long: `Fuzzy-finds a note by filename and opens it in your editor.

Examples:
  vault open my-note
  vault open meeting-notes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, notePath, err := mgr.OpenNote(name)
		if err != nil {
			return fmt.Errorf("open note: %w", err)
		}

		fullPath := notePath
		if !filepath.IsAbs(notePath) {
			fullPath = filepath.Join(mgr.NotesDir(), notePath)
		}

		if err := openInEditor(fullPath); err != nil {
			return fmt.Errorf("open editor: %w", err)
		}
		_ = note
		return nil
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
