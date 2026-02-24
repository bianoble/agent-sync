package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Build-time variables set via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Global flags.
var (
	configPath   string
	lockfilePath string
	verbose      bool
	quiet        bool
	noColor      bool
)

var rootCmd = &cobra.Command{
	Use:   "agent-sync",
	Short: "Deterministic synchronization of agent files",
	Long: `agent-sync is a deterministic, registry-agnostic synchronization system
for agent files. It fetches files from external sources (Git, URL, local),
pins them immutably via a lockfile, applies controlled transformations,
and synchronizes them into tool-specific project locations.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("agent-sync %s\n", version)
		fmt.Printf("  commit:  %s\n", commit)
		fmt.Printf("  built:   %s\n", date)
		fmt.Printf("  spec:    v1\n")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "agent-sync.yaml", "path to config file")
	rootCmd.PersistentFlags().StringVar(&lockfilePath, "lockfile", "agent-sync.lock", "path to lockfile")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "detailed output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "minimal output (errors only)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
