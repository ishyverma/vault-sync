package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Open or create today's daily note",
	Long: `Opens or creates the daily note for today's date using the daily template.
The note is named YYYY-MM-DD.md.

Examples:
  vault daily`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		today := time.Now().Format("2006-01-02")
		name := fmt.Sprintf("%s.md", today)

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, notePath, err := mgr.OpenNote(name)
		if err != nil {
			note, err = mgr.CreateNote(name, "daily")
			if err != nil {
				return fmt.Errorf("create daily note: %w", err)
			}
			notePath = filepath.Join(mgr.NotesDir(), note.Filename)
		}

		fmt.Printf("✓ Daily note: %s\n", note.Filename)
		if err := openInEditor(notePath); err != nil {
			return fmt.Errorf("open editor: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dailyCmd)
}
