package main

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var quickCmd = &cobra.Command{
	Use:   "quick",
	Short: "Open a quick scratch buffer",
	Long: `Opens a temporary scratch note for quick capture.
The note is auto-saved and indexed when you exit the editor.

Examples:
  vault quick`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := make([]byte, 4)
		rand.Read(b)
		suffix := fmt.Sprintf("%x", b)
		today := time.Now().Format("2006-01-02")
		name := fmt.Sprintf("scratch-%s-%s", today, suffix)

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := mgr.CreateNote(name, "blank")
		if err != nil {
			return fmt.Errorf("create scratch note: %w", err)
		}

		notePath := filepath.Join(mgr.NotesDir(), note.Filename)
		fmt.Printf("Scratch note: %s\n", note.Filename)

		if err := openInEditor(notePath); err != nil {
			return fmt.Errorf("open editor: %w", err)
		}

		if err := mgr.SyncFromDisk(note.ID); err != nil {
			return fmt.Errorf("sync scratch note: %w", err)
		}

		fmt.Printf("✓ Saved scratch note: %s\n", note.Filename)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(quickCmd)
}
