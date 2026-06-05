package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new note and open it in your editor",
	Long: `Creates a new markdown note in your vault and opens it in Vim.
You can optionally specify a template with --template.

Examples:
  vault new my-note
  vault new meeting-notes --template meeting
  vault new work/todo --template project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		template, _ := cmd.Flags().GetString("template")

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := mgr.CreateNote(name, template)
		if err != nil {
			return fmt.Errorf("create note: %w", err)
		}

		notePath := filepath.Join(mgr.NotesDir(), note.Filename)
		fmt.Printf("✓ Created %s\n", notePath)

		open := true
		if f, _ := cmd.Flags().GetBool("no-open"); f {
			open = false
		}

		if open {
			if err := openInEditor(notePath); err != nil {
				return fmt.Errorf("open editor: %w", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringP("template", "t", "blank", "Template to use (blank, daily, meeting, project)")
	newCmd.Flags().Bool("no-open", false, "Create note without opening editor")
}
