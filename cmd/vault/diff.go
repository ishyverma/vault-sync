package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [note-name]",
	Short: "Show differences between note versions",
	Long: `Displays a line-by-line diff between two versions of a note.
Uses --v1 and --v2 to specify versions (default: last two).

Examples:
  vault diff my-note
  vault diff my-note --v1 1 --v2 3`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		note, err := findNote(mgr, name)
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		versions, err := store.ListVersions(note.ID)
		if err != nil {
			return fmt.Errorf("list versions: %w", err)
		}

		if len(versions) < 2 {
			return fmt.Errorf("not enough versions to diff (need at least 2)")
		}

		v1, _ := cmd.Flags().GetInt("v1")
		v2, _ := cmd.Flags().GetInt("v2")

		if v1 == 0 {
			v1 = versions[len(versions)-2].VersionNum
		}
		if v2 == 0 {
			v2 = versions[len(versions)-1].VersionNum
		}

		ver1, err := store.GetVersion(note.ID, v1)
		if err != nil {
			return fmt.Errorf("get version %d: %w", v1, err)
		}
		ver2, err := store.GetVersion(note.ID, v2)
		if err != nil {
			return fmt.Errorf("get version %d: %w", v2, err)
		}

		fmt.Printf("Diff %s (v%d → v%d):\n\n", note.Filename, v1, v2)
		printDiff(ver1.Content, ver2.Content)
		return nil
	},
}

func printDiff(old, new string) {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	max := len(oldLines)
	if len(newLines) > max {
		max = len(newLines)
	}

	for i := 0; i < max; i++ {
		switch {
		case i >= len(oldLines):
			fmt.Printf("+ %s\n", newLines[i])
		case i >= len(newLines):
			fmt.Printf("- %s\n", oldLines[i])
		case oldLines[i] != newLines[i]:
			fmt.Printf("- %s\n", oldLines[i])
			fmt.Printf("+ %s\n", newLines[i])
		}
	}
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().Int("v1", 0, "First version number (default: second-to-last)")
	diffCmd.Flags().Int("v2", 0, "Second version number (default: latest)")
}
