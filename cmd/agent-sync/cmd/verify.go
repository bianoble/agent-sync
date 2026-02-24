package cmd

import (
	"fmt"

	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [source-name...]",
	Short: "Verify the lockfile against upstream sources",
	Long: `Checks whether upstream sources have changed since the lockfile was last written.
Reports which sources have newer content available. Does NOT modify the lockfile
or target files. Exit 0 if all sources match; exit non-zero if changes are available.`,
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

		eng := &engine.VerifyEngine{
			Registry:    newRegistry(),
			ProjectRoot: root,
		}

		result, err := eng.Verify(cmd.Context(), *lf, *cfg, args)
		if err != nil {
			return err
		}

		for _, name := range result.UpToDate {
			info("  ✓ %-20s  up to date", name)
		}
		for _, d := range result.Changed {
			info("  ✗ %-20s  %s → %s", d.Source, d.Before, d.After)
		}
		for _, e := range result.Errors {
			errorf("%s: %s", e.Source, e.Err)
		}

		if len(result.Changed) > 0 || len(result.Errors) > 0 {
			return fmt.Errorf("%d source(s) have upstream changes", len(result.Changed)+len(result.Errors))
		}

		info("\nAll sources match upstream.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
