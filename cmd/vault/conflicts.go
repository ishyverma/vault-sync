package main

import (
	"fmt"
	"os"

	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/sync"
	"github.com/spf13/cobra"
)

var conflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "List and resolve sync conflicts",
	Long: `Shows all notes with sync conflicts (local and remote both diverged).
Use --resolve to auto-resolve all with a strategy.

  vault conflicts                    List all conflicted notes
  vault conflicts --resolve local    Auto-resolve keeping local version
  vault conflicts --resolve remote   Auto-resolve keeping remote version`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := newSyncEngine()
		if err != nil {
			return fmt.Errorf("sync engine: %w", err)
		}

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		strategy, _ := cmd.Flags().GetString("resolve")
		if strategy != "" {
			return resolveConflicts(engine, mgr, strategy)
		}
		return listConflicts(engine, mgr)
	},
}

func listConflicts(engine *sync.Engine, mgr *core.Manager) error {
	states, err := engine.ListConflicts()
	if err != nil {
		return fmt.Errorf("list conflicts: %w", err)
	}
	if len(states) == 0 {
		fmt.Println("✓ No conflicts detected")
		return nil
	}

	fmt.Printf("Found %d conflict(s):\n\n", len(states))
	fmt.Printf("%-30s %-10s %s\n", "NOTE", "BACKEND", "ERROR")
	fmt.Println("──────────────────────────────────────────────────────")
	for _, s := range states {
		note, _ := mgr.GetNote(s.NoteID)
		name := s.NoteID
		if note != nil {
			name = note.Filename
		}
		if len(name) > 29 {
			name = name[:26] + "..."
		}
		errMsg := s.ErrorMsg
		if errMsg == "" {
			errMsg = "remote file modified externally"
		}
		fmt.Printf("%-30s %-10s %s\n", name, s.Backend, errMsg)
	}
	fmt.Println()
	fmt.Println("Resolve with: vault conflicts --resolve local|remote")
	return nil
}

func resolveConflicts(engine *sync.Engine, mgr *core.Manager, strategy string) error {
	if strategy != "local" && strategy != "remote" {
		return fmt.Errorf("strategy must be 'local' or 'remote', got '%s'", strategy)
	}

	states, err := engine.ListConflicts()
	if err != nil {
		return fmt.Errorf("list conflicts: %w", err)
	}
	if len(states) == 0 {
		fmt.Println("✓ No conflicts to resolve")
		return nil
	}

	var resolved, failed int
	for _, s := range states {
		if err := engine.ResolveConflict(s.NoteID, s.Backend, strategy); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s/%s: %v\n", s.NoteID, s.Backend, err)
			failed++
		} else {
			note, _ := mgr.GetNote(s.NoteID)
			name := s.NoteID
			if note != nil {
				name = note.Filename
			}
			fmt.Printf("✓ %s/%s resolved (%s wins)\n", name, s.Backend, strategy)
			resolved++
		}
	}

	fmt.Printf("\n%d resolved, %d failed\n", resolved, failed)
	return nil
}

func init() {
	rootCmd.AddCommand(conflictsCmd)
	conflictsCmd.Flags().String("resolve", "", "Auto-resolve all conflicts (local|remote)")
}
