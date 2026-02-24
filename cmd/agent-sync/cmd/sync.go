package cmd

import (
	"fmt"

	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/spf13/cobra"
)

var syncDryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize files to targets using the lockfile",
	Long: `Reads the lockfile as the source of truth, fetches content from cache or
sources as needed, and writes files to target locations. Does NOT modify the
lockfile — only 'update' and 'prune' modify the lockfile.`,
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

		c, err := newCache()
		if err != nil {
			return err
		}

		eng := &engine.SyncEngine{
			Registry:    newRegistry(),
			Cache:       c,
			ToolMap:     newToolMap(cfg),
			ProjectRoot: root,
		}

		opts := engine.SyncOptions{DryRun: syncDryRun}
		result, err := eng.Sync(cmd.Context(), *lf, *cfg, opts)
		if err != nil {
			return err
		}

		if syncDryRun {
			info("Dry run — no files written.")
		}

		for _, f := range result.Written {
			info("  %s  %s", f.Action, f.Path)
		}
		for _, f := range result.Skipped {
			detail("  %s  %s", f.Action, f.Path)
		}
		for _, e := range result.Errors {
			errorf("%s: %s", e.Source, e.Err)
		}

		total := len(result.Written) + len(result.Skipped)
		info("")
		info("Sync complete: %d written, %d unchanged, %d errors.",
			len(result.Written), len(result.Skipped), len(result.Errors))

		_ = total
		if len(result.Errors) > 0 {
			return fmt.Errorf("%d source(s) failed", len(result.Errors))
		}
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "show what would change without writing files")
	rootCmd.AddCommand(syncCmd)
}
