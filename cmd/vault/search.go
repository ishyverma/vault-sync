package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search notes by title or filename",
	Long: `Searches across all notes and displays matching results.

Examples:
  vault search golang
  vault search "meeting notes"
  vault search --tag work`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		notes, err := mgr.SearchNotes(query)
		if err != nil {
			return fmt.Errorf("search notes: %w", err)
		}

		if len(notes) == 0 {
			fmt.Printf("No results for %q\n", query)
			return nil
		}

		fmt.Printf("Found %d result(s) for %q:\n\n", len(notes), query)
		fmt.Printf("%-30s %-20s %-10s\n", "NAME", "TITLE", "TAGS")
		fmt.Println("──────────────────────────────────────────────────────────")
		for _, n := range notes {
			name := n.Filename
			if len(name) > 29 {
				name = name[:26] + "..."
			}
			title := n.Title
			if len(title) > 19 {
				title = title[:16] + "..."
			}
			fmt.Printf("%-30s %-20s %-10s\n", name, title, formatTags(n.Tags))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
