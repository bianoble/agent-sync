package cmd

import (
	"fmt"

	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/spf13/cobra"
)

var pruneDryRun bool

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove files no longer referenced in the configuration",
	Long: `Compares current config targets against files tracked in the lockfile.
Removes files that were previously synced but are no longer in the config.
Use --dry-run to see what would be removed without acting.`,
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

		eng := &engine.PruneEngine{
			ToolMap:     newToolMap(cfg),
			ProjectRoot: root,
		}

		opts := engine.PruneOptions{DryRun: pruneDryRun}
		result, err := eng.Prune(cmd.Context(), *lf, *cfg, opts)
		if err != nil {
			return err
		}

		if pruneDryRun {
			info("Dry run — no files removed.")
		}

		if len(result.Removed) == 0 {
			info("Nothing to prune.")
			return nil
		}

		for _, f := range result.Removed {
			info("  %s  %s", f.Action, f.Path)
		}
		info("\nPruned %d file(s).", len(result.Removed))

		if len(result.Errors) > 0 {
			for _, e := range result.Errors {
				errorf("%s: %s", e.Source, e.Err)
			}
			return fmt.Errorf("%d error(s) during prune", len(result.Errors))
		}
		return nil
	},
}

func init() {
	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "show what would be removed without acting")
	rootCmd.AddCommand(pruneCmd)
}
