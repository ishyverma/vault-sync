package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var backlinksCmd = &cobra.Command{
	Use:   "backlinks [note-name]",
	Short: "Find notes linking to this note",
	Long: `Scans all notes for [[WikiLink]] references to the given note.

Examples:
  vault backlinks my-note
  vault backlinks my-note --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := findNote(mgr, args[0])
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		refName := strings.TrimSuffix(note.Filename, ".md")
		notes, err := mgr.ListNotes()
		if err != nil {
			return fmt.Errorf("list notes: %w", err)
		}

		var backlinks []string
		for _, n := range notes {
			if n.ID == note.ID {
				continue
			}
			content, readErr := os.ReadFile(filepath.Join(mgr.NotesDir(), n.Filename))
			if readErr != nil {
				continue
			}
			pattern := "[[" + refName
			if strings.Contains(string(content), pattern) {
				backlinks = append(backlinks, n.Filename)
			}
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(backlinks)
		}

		if len(backlinks) == 0 {
			fmt.Printf("No backlinks to %s\n", note.Filename)
			return nil
		}

		fmt.Printf("Notes linking to %s:\n\n", note.Filename)
		for _, b := range backlinks {
			fmt.Printf("  ← %s\n", b)
		}
		return nil
	},
}

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Show ASCII note connection graph",
	Long: `Displays an ASCII graph of WikiLink connections between notes.

Examples:
  vault graph
  vault graph --json`,
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

		links := make(map[string][]string)
		for _, n := range notes {
			content, readErr := os.ReadFile(filepath.Join(mgr.NotesDir(), n.Filename))
			if readErr != nil {
				continue
			}
			text := string(content)
			for _, other := range notes {
				if other.ID == n.ID {
					continue
				}
				refName := strings.TrimSuffix(other.Filename, ".md")
				if strings.Contains(text, "[["+refName) {
					links[n.Filename] = append(links[n.Filename], other.Filename)
				}
			}
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(links)
		}

		if len(links) == 0 {
			fmt.Println("No connections found between notes")
			return nil
		}

		fmt.Println("Note Graph (WikiLink connections):")
		fmt.Println(strings.Repeat("─", 60))
		for from, targets := range links {
			for _, to := range targets {
				fmt.Printf("  %s → %s\n", from, to)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backlinksCmd)
	rootCmd.AddCommand(graphCmd)
	backlinksCmd.Flags().Bool("json", false, "Output as JSON")
	graphCmd.Flags().Bool("json", false, "Output as JSON")
}
