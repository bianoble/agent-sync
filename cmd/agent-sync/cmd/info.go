package cmd

import (
	"fmt"

	"github.com/bianoble/agent-sync/internal/engine"
	"github.com/bianoble/agent-sync/internal/target"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show information about agent-sync configuration and tools",
	Long: `Displays the agent-sync version, spec version, configuration and lockfile paths,
cache directory and size, and known tool definitions (built-in and custom).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := loadConfig() // ok if config doesn't exist
		c, _ := newCache()

		var tm *target.ToolMap
		if cfg != nil {
			tm = newToolMap(cfg)
		}

		result, err := engine.Info(version, cfg, c, tm, configPath, lockfilePath)
		if err != nil {
			return err
		}

		fmt.Printf("agent-sync %s\n", result.Version)
		fmt.Printf("  spec version:  %d\n", result.SpecVersion)
		fmt.Printf("  config:        %s\n", result.ConfigPath)
		fmt.Printf("  lockfile:      %s\n", result.LockPath)
		fmt.Printf("  cache dir:     %s\n", result.CacheDir)
		fmt.Printf("  cache size:    %s\n", humanSize(result.CacheSize))

		if len(result.Tools) > 0 {
			fmt.Println("\nTool definitions:")
			for _, t := range result.Tools {
				custom := ""
				if t.IsCustom {
					custom = " (custom)"
				}
				fmt.Printf("  %-15s → %s%s\n", t.Name, t.Destination, custom)
			}
		}

		return nil
	},
}

func humanSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
