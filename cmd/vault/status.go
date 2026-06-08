package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/connectors/git"
	"github.com/ishyverma/vault-sync/internal/connectors/notion"
	"github.com/ishyverma/vault-sync/internal/connectors/obsidian"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show connection status for all backends",
	Long: `Displays the status of each configured sync backend.
Also shows overall vault statistics.

Examples:
  vault status
  vault status --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		mgr, err := newManager()
		if err != nil {
			return fmt.Errorf("open vault: %w", err)
		}

		notes, _ := mgr.ListNotes()
		engine, _ := newSyncEngine()
		queueLen, _ := engine.QueueLength()
		conflicts, _ := engine.ListConflicts()

		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			backends := map[string]interface{}{}
			if cfg.Backends.Notion.Enabled {
				conn := notion.NewConnector(cfg.Backends.Notion.Token, cfg.Backends.Notion.TargetPageID, cfg.Backends.Notion.DatabaseID, "")
				healthy, err := conn.Status()
				backends["notion"] = map[string]interface{}{"enabled": true, "healthy": healthy, "error": errStr(err)}
			} else {
				backends["notion"] = map[string]interface{}{"enabled": false}
			}
			if cfg.Backends.Obsidian.Enabled {
				conn := obsidian.NewConnector(cfg.Backends.Obsidian.VaultPath, cfg.Backends.Obsidian.Subfolder, "", cfg.Backends.Obsidian.Wikilinks)
				healthy, err := conn.Status()
				backends["obsidian"] = map[string]interface{}{"enabled": true, "healthy": healthy, "error": errStr(err)}
			} else {
				backends["obsidian"] = map[string]interface{}{"enabled": false}
			}
			if cfg.Backends.Git.Enabled {
				conn := git.NewConnector(cfg.Backends.Git.RepoPath, cfg.Backends.Git.CommitMessage, cfg.Backends.Git.Remote, cfg.Backends.Git.AutoCommit)
				healthy, err := conn.Status()
				backends["git"] = map[string]interface{}{"enabled": true, "healthy": healthy, "error": errStr(err)}
			} else {
				backends["git"] = map[string]interface{}{"enabled": false}
			}
			out := map[string]interface{}{
				"notes":     len(notes),
				"queue":     queueLen,
				"conflicts": len(conflicts),
				"backends":  backends,
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}

		fmt.Println("VaultSync Status")
		fmt.Println(strings.Repeat("─", 50))
		fmt.Printf("Notes:     %d\n", len(notes))
		fmt.Printf("Queue:     %d pending\n", queueLen)
		fmt.Printf("Conflicts: %d\n", len(conflicts))
		fmt.Println()

		fmt.Println("Backends:")
		fmt.Println(strings.Repeat("─", 50))

		printBackendStatus(cfg.Backends.Notion.Enabled, "Notion", func() (bool, error) {
			if !cfg.Backends.Notion.Enabled {
				return false, nil
			}
			conn := notion.NewConnector(cfg.Backends.Notion.Token, cfg.Backends.Notion.TargetPageID, cfg.Backends.Notion.DatabaseID, "")
			return conn.Status()
		})
		printBackendStatus(cfg.Backends.Obsidian.Enabled, "Obsidian", func() (bool, error) {
			if !cfg.Backends.Obsidian.Enabled {
				return false, nil
			}
			conn := obsidian.NewConnector(cfg.Backends.Obsidian.VaultPath, cfg.Backends.Obsidian.Subfolder, "", cfg.Backends.Obsidian.Wikilinks)
			return conn.Status()
		})
		printBackendStatus(cfg.Backends.Git.Enabled, "Git", func() (bool, error) {
			if !cfg.Backends.Git.Enabled {
				return false, nil
			}
			conn := git.NewConnector(cfg.Backends.Git.RepoPath, cfg.Backends.Git.CommitMessage, cfg.Backends.Git.Remote, cfg.Backends.Git.AutoCommit)
			return conn.Status()
		})

		return nil
	},
}

func printBackendStatus(enabled bool, name string, check func() (bool, error)) {
	if !enabled {
		fmt.Printf("  ○ %-10s not configured\n", name)
		return
	}
	healthy, err := check()
	if healthy {
		fmt.Printf("  ● %-10s healthy\n", name)
	} else if err != nil {
		fmt.Printf("  ✗ %-10s error: %v\n", name, err)
	} else {
		fmt.Printf("  ● %-10s connected\n", name)
	}
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status for all notes",
	Long: `Displays the sync status of each note across all connected backends.

The status column shows:
  synced     — Note is up-to-date with the backend
  pending    — Note is queued for sync
  conflict   — Remote has diverged from local
  failed     — Last sync attempt failed
  local_only — Note has never been synced to this backend`,
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

		notes, err := mgr.ListNotes()
		if err != nil {
			return fmt.Errorf("list notes: %w", err)
		}

		if len(notes) == 0 {
			fmt.Println("No notes found. Create one with: vault new <name>")
			return nil
		}

		queueLen, _ := engine.QueueLength()
		asJSON, _ := cmd.Flags().GetBool("json")

		if asJSON {
			var statusRows []map[string]interface{}
			for _, n := range notes {
				states, err := engine.SyncStatus(n.ID)
				if err != nil || len(states) == 0 {
					continue
				}
				for _, s := range states {
					statusRows = append(statusRows, map[string]interface{}{
						"note":       n.Filename,
						"backend":    s.Backend,
						"status":     s.Status,
						"last_sync":  s.LastSyncAt,
						"error":      s.ErrorMsg,
						"remote_id":  s.RemoteID,
					})
				}
			}
			out := map[string]interface{}{
				"queue_length": queueLen,
				"entries":      statusRows,
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}

		fmt.Printf("%-30s %-12s %s\n", "NOTE", "STATUS", "LAST SYNC")
		fmt.Println("──────────────────────────────────────────────────────────────")
		for _, n := range notes {
			states, err := engine.SyncStatus(n.ID)
			if err != nil || len(states) == 0 {
				fmt.Printf("%-30s %-12s %s\n", n.Filename, "—", "—")
				continue
			}
			for _, s := range states {
				lastSync := "never"
				if !s.LastSyncAt.IsZero() {
					lastSync = fmtDuration(time.Since(s.LastSyncAt)) + " ago"
				}
				name := n.Filename
				if len(name) > 29 {
					name = name[:26] + "..."
				}
				label := s.Backend + ":" + s.Status
				fmt.Printf("%-30s %-12s %s\n", name, label, lastSync)
			}
		}

		if queueLen > 0 {
			fmt.Printf("\n⚠ %d pending sync job(s) in queue\n", queueLen)
		}

		return nil
	},
}

func fmtDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m"
		}
		return fmt.Sprintf("%dm", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h"
		}
		return fmt.Sprintf("%dh", h)
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
	syncCmd.AddCommand(syncStatusCmd)
	statusCmd.Flags().Bool("json", false, "Output as JSON")
	syncStatusCmd.Flags().Bool("json", false, "Output as JSON")
}
