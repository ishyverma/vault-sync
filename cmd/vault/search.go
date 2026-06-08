package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search notes by title or filename",
	Long: `Searches across all notes and displays matching results.

Examples:
  vault search golang
  vault search "meeting notes"
  vault search --tag work
  vault search --after 2024-01-01 --before 2024-06-01
  vault search --regex "meeting.*q[123]"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		tag, _ := cmd.Flags().GetString("tag")
		afterStr, _ := cmd.Flags().GetString("after")
		beforeStr, _ := cmd.Flags().GetString("before")
		regexPattern, _ := cmd.Flags().GetString("regex")

		var afterDate, beforeDate time.Time
		if afterStr != "" {
			afterDate, err = time.Parse("2006-01-02", afterStr)
			if err != nil {
				return fmt.Errorf("invalid --after date (use YYYY-MM-DD): %w", err)
			}
		}
		if beforeStr != "" {
			beforeDate, err = time.Parse("2006-01-02", beforeStr)
			if err != nil {
				return fmt.Errorf("invalid --before date (use YYYY-MM-DD): %w", err)
			}
		}

		var re *regexp.Regexp
		if regexPattern != "" {
			re, err = regexp.Compile(regexPattern)
			if err != nil {
				return fmt.Errorf("invalid --regex pattern: %w", err)
			}
		}

		var notes []*storage.Note
		if tag != "" {
			notes, err = mgr.ListNotesByTag(tag)
			if err != nil {
				return fmt.Errorf("search by tag: %w", err)
			}
		} else if len(args) > 0 {
			query := args[0]
			notes, err = mgr.SearchNotes(query)
			if err != nil {
				return fmt.Errorf("search notes: %w", err)
			}
		} else {
			notes, err = mgr.ListNotes()
			if err != nil {
				return fmt.Errorf("list notes: %w", err)
			}
		}

		notes = filterNotes(notes, afterDate, beforeDate, re)

		if len(notes) == 0 {
			fmt.Printf("No results\n")
			return nil
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(notes)
		}

		fmt.Printf("Found %d result(s):\n\n", len(notes))
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

func filterNotes(notes []*storage.Note, after, before time.Time, re *regexp.Regexp) []*storage.Note {
	var filtered []*storage.Note
	for _, n := range notes {
		if !after.IsZero() && n.CreatedAt.Before(after) {
			continue
		}
		if !before.IsZero() && n.CreatedAt.After(before) {
			continue
		}
		if re != nil && !re.MatchString(n.Title) && !re.MatchString(n.Filename) {
			continue
		}
		filtered = append(filtered, n)
	}
	return filtered
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().Bool("json", false, "Output as JSON")
	searchCmd.Flags().String("tag", "", "Filter by tag")
	searchCmd.Flags().String("after", "", "Filter notes created after this date (YYYY-MM-DD)")
	searchCmd.Flags().String("before", "", "Filter notes created before this date (YYYY-MM-DD)")
	searchCmd.Flags().String("regex", "", "Filter by regex pattern on title/filename")
}
