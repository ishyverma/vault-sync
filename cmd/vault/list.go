package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes in your vault",
	Long: `Displays a table of all notes with their metadata.

Examples:
  vault list
  vault list --tag work`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		tag, _ := cmd.Flags().GetString("tag")
		asJSON, _ := cmd.Flags().GetBool("json")

		var notes []NoteRow
		if tag != "" {
			allNotes, err := mgr.ListNotes()
			if err != nil {
				return fmt.Errorf("list notes: %w", err)
			}
			for _, n := range allNotes {
				for _, t := range n.Tags {
					if t == tag {
						notes = append(notes, NoteRow{n.Filename, n.Title, formatTags(n.Tags), n.WordCount, n.CreatedAt.Format("2006-01-02")})
						break
					}
				}
			}
		} else {
			allNotes, err := mgr.ListNotes()
			if err != nil {
				return fmt.Errorf("list notes: %w", err)
			}
			for _, n := range allNotes {
				notes = append(notes, NoteRow{n.Filename, n.Title, formatTags(n.Tags), n.WordCount, n.CreatedAt.Format("2006-01-02")})
			}
		}

		if len(notes) == 0 {
			fmt.Println("No notes found. Create one with: vault new <name>")
			return nil
		}

		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(notes)
		}

		fmt.Printf("%-30s %-20s %-10s\n", "NAME", "TITLE", "WORDS")
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
			fmt.Printf("%-30s %-20s %5d\n", name, title, n.Words)
		}
		return nil
	},
}

type NoteRow struct {
	Filename string
	Title    string
	Tags     string
	Words    int
	Date     string
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	s := ""
	for i, t := range tags {
		if i > 0 {
			s += ", "
		}
		s += t
	}
	return s
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("tag", "t", "", "Filter notes by tag")
	listCmd.Flags().Bool("json", false, "Output as JSON")
}
