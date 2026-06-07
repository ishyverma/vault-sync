package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history [note-name]",
	Short: "View version history for a note",
	Long: `Lists all saved versions of a note with timestamps and triggers.

Examples:
  vault history my-note
  vault history my-note --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		note, err := findNote(mgr, args[0])
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		versions, err := store.ListVersions(note.ID)
		if err != nil {
			return fmt.Errorf("list versions: %w", err)
		}

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(versions)
		}

		if len(versions) == 0 {
			fmt.Printf("No version history for %s\n", note.Filename)
			return nil
		}

		fmt.Printf("Version history for %s:\n\n", note.Filename)
		fmt.Printf("%-8s %-22s %-12s %s\n", "VERSION", "SAVED AT", "TRIGGER", "PREVIEW")
		fmt.Println("────────────────────────────────────────────────────────────")
		for _, v := range versions {
			preview := v.Content
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			preview = stripNewlines(preview)
			fmt.Printf("%-8d %-22s %-12s %s\n", v.VersionNum, v.SavedAt.Format(time.RFC3339), v.Trigger, preview)
		}
		fmt.Println()
		fmt.Println("Restore with: vault restore <note> --version <num>")
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore [note-name]",
	Short: "Restore a note to a previous version",
	Long: `Restores a note to a specific version from its history.

Examples:
  vault restore my-note --version 3
  vault restore my-note --version 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		store, err := newStore()
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		versionNum, _ := cmd.Flags().GetInt("version")
		if versionNum < 1 {
			return fmt.Errorf("--version must be >= 1")
		}

		note, err := findNote(mgr, args[0])
		if err != nil {
			return fmt.Errorf("find note: %w", err)
		}

		v, err := store.GetVersion(note.ID, versionNum)
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		localPath := filepath.Join(mgr.NotesDir(), note.Filename)
		if err := os.WriteFile(localPath, []byte(v.Content), 0o644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}

		store.SaveVersion(note.ID, v.Content, "restore")

		fmt.Printf("✓ Restored %s to version %d\n", note.Filename, versionNum)
		return nil
	},
}

func newStore() (*storage.NoteStore, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	vaultDir := resolveVaultDir(cfg)
	store := storage.NewNoteStore(vaultDir)
	if err := store.Init(); err != nil {
		return nil, err
	}
	return store, nil
}

func findNote(mgr *core.Manager, name string) (*storage.Note, error) {
	notes, err := mgr.ListNotes()
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		if n.Filename == name || n.Filename == name+".md" || n.Title == name {
			return n, nil
		}
	}
	return nil, fmt.Errorf("note not found: %s", name)
}

func stripNewlines(s string) string {
	if len(s) > 60 && s[59] == '\n' {
		return s[:57] + "..."
	}
	var out []byte
	for _, b := range []byte(s) {
		if b == '\n' || b == '\r' {
			out = append(out, ' ')
		} else {
			out = append(out, b)
		}
	}
	return string(out)
}

func init() {
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(restoreCmd)
	historyCmd.Flags().Bool("json", false, "Output as JSON")
	restoreCmd.Flags().IntP("version", "n", 0, "Version number to restore")
	restoreCmd.MarkFlagRequired("version")
}
