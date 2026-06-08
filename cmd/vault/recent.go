package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show recently modified notes",
	Long: `Displays notes sorted by most recently modified, limited to
the last 10 notes by default. Use --limit to change.

Examples:
  vault recent
  vault recent --limit 20
  vault recent --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		notes, err := mgr.ListNotes()
		if err != nil {
			return fmt.Errorf("list notes: %w", err)
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		sort.Slice(notes, func(i, j int) bool {
			return notes[i].ModifiedAt.After(notes[j].ModifiedAt)
		})

		limit, _ := cmd.Flags().GetInt("limit")
		if limit > 0 && limit < len(notes) {
			notes = notes[:limit]
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			type row struct {
				Filename   string `json:"filename"`
				Title      string `json:"title"`
				ModifiedAt string `json:"modified_at"`
				Words      int    `json:"words"`
			}
			var rows []row
			for _, n := range notes {
				rows = append(rows, row{n.Filename, n.Title, n.ModifiedAt.Format(time.RFC3339), n.WordCount})
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
		}

		fmt.Printf("%-30s %-20s %s\n", "NAME", "TITLE", "MODIFIED")
		fmt.Println("──────────────────────────────────────────────────────────────")
		for _, n := range notes {
			name := n.Filename
			if len(name) > 29 {
				name = name[:26] + "..."
			}
			title := n.Title
			if len(title) > 19 {
				title = title[:16] + "..."
			}
			fmt.Printf("%-30s %-20s %s ago\n", name, title, fmtDuration(time.Since(n.ModifiedAt)))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(recentCmd)
	recentCmd.Flags().IntP("limit", "n", 10, "Maximum number of notes to show")
	recentCmd.Flags().Bool("json", false, "Output as JSON")
}
