package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/bianoble/agent-sync/internal/lock"
	"github.com/spf13/cobra"
)

var (
	updateDryRun bool
	updateYes    bool
)

var updateCmd = &cobra.Command{
	Use:   "update [source-name...]",
	Short: "Resolve sources against upstream and update the lockfile",
	Long: `Resolves each source to its current upstream state, shows a diff of lockfile
changes, and updates the lockfile. If source names are provided, only those
sources are updated; others are left unchanged.`,
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

		eng := &engine.UpdateEngine{
			Registry:    newRegistry(),
			Cache:       c,
			ProjectRoot: root,
		}

		opts := engine.UpdateOptions{
			DryRun:      updateDryRun,
			AutoConfirm: updateYes,
			SourceNames: args,
		}

		result, err := eng.Update(cmd.Context(), *cfg, lf, opts)
		if err != nil {
			return err
		}

		// Display changes.
		if len(result.Updated) == 0 && len(result.Failed) == 0 {
			info("All sources are up to date.")
			return nil
		}

		for _, u := range result.Updated {
			before := "(new)"
			if u.Before != nil {
				before = summarizeLockedSource(u.Before)
			}
			after := summarizeLockedSource(u.After)
			info("  %-20s  %s → %s", u.Name, before, after)
		}
		for _, e := range result.Failed {
			errorf("%s: %s", e.Source, e.Err)
		}

		if updateDryRun {
			info("\nDry run — lockfile not modified.")
			return nil
		}

		// Confirm unless --yes.
		if !updateYes && len(result.Updated) > 0 {
			fmt.Printf("\nApply %d update(s) to lockfile? [y/N] ", len(result.Updated))
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					info("Aborted.")
					return nil
				}
			}
		}

		// Save lockfile.
		if result.Lockfile != nil {
			if err := saveLockfile(result.Lockfile); err != nil {
				return fmt.Errorf("saving lockfile: %w", err)
			}
			info("\nLockfile updated.")
		}

		if len(result.Failed) > 0 {
			return fmt.Errorf("%d source(s) failed to resolve", len(result.Failed))
		}
		return nil
	},
}

func summarizeLockedSource(ls *lock.LockedSource) string {
	if ls == nil {
		return "(none)"
	}
	if ls.Resolved.Commit != "" {
		short := ls.Resolved.Commit
		if len(short) > 8 {
			short = short[:8]
		}
		return short
	}
	if ls.Resolved.SHA256 != "" {
		short := ls.Resolved.SHA256
		if len(short) > 8 {
			short = short[:8]
		}
		return "sha256:" + short
	}
	if len(ls.Resolved.Files) > 0 {
		return fmt.Sprintf("(%d files)", len(ls.Resolved.Files))
	}
	return "(unknown)"
}

func init() {
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "show what would change without updating the lockfile")
	updateCmd.Flags().BoolVar(&updateYes, "yes", false, "skip interactive confirmation")
	rootCmd.AddCommand(updateCmd)
}
