package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

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
	syncCmd.AddCommand(syncStatusCmd)
	syncStatusCmd.Flags().Bool("json", false, "Output as JSON")
}
