package cmd

import (
	"fmt"

	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify that target files match the lockfile",
	Long: `Hashes all target files and compares them against the lockfile.
Reports any drift (files changed, missing, or unexpected).
Exit 0 if everything matches; exit non-zero on drift. Suitable for CI pipelines.`,
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

		eng := &engine.CheckEngine{
			ToolMap:     newToolMap(cfg),
			ProjectRoot: root,
		}

		result, err := eng.Check(cmd.Context(), *lf, *cfg)
		if err != nil {
			return err
		}

		if result.Clean {
			info("All files match the lockfile.")
			return nil
		}

		for _, d := range result.Drifted {
			info("  drifted   %s", d.Path)
			detail("expected: %s", d.Expected)
			detail("actual:   %s", d.Actual)
		}
		for _, m := range result.Missing {
			info("  missing   %s", m)
		}

		total := len(result.Drifted) + len(result.Missing)
		return fmt.Errorf("check failed: %d file(s) out of sync", total)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
