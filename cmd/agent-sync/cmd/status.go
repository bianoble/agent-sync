package cmd

import (
	"fmt"
	"strings"

	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [source-name...]",
	Short: "Show the current state of all synced sources",
	Long: `Shows source name, type, pinned version/commit, target destinations,
and sync state (synced, drifted, missing, pending) for all or named sources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		lf, err := loadLockfile()
		if err != nil {
			return err
		}

		root, err := projectRoot()
		if err != nil {
			return err
		}

		eng := &engine.StatusEngine{
			ToolMap:     newToolMap(cfg),
			ProjectRoot: root,
		}

		statuses, err := eng.Status(cmd.Context(), *lf, *cfg, args)
		if err != nil {
			return err
		}

		if len(statuses) == 0 {
			info("No sources configured.")
			return nil
		}

		// Print table header.
		fmt.Printf("%-20s %-8s %-16s %-30s %s\n", "SOURCE", "TYPE", "PINNED AT", "TARGETS", "STATE")
		for _, s := range statuses {
			targets := strings.Join(s.Targets, ", ")
			if len(targets) > 30 {
				targets = targets[:27] + "..."
			}
			fmt.Printf("%-20s %-8s %-16s %-30s %s\n", s.Name, s.Type, s.PinnedAt, targets, s.State)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
