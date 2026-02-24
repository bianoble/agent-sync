package cmd

import (
	"fmt"

	"github.com/bianoble/agent-sync/internal/config"
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
		// Load config with hierarchy to capture layer metadata.
		hr, _ := loadConfigHierarchical() // ok if config doesn't exist
		var cfg *config.Config
		if hr != nil {
			cfg = hr.Config
		}
		c, _ := newCache()

		var tm *target.ToolMap
		if cfg != nil {
			tm = newToolMap(cfg)
		}

		result, err := engine.Info(version, cfg, c, tm, configPath, lockfilePath)
		if err != nil {
			return err
		}

		// Populate config chain from hierarchical load result.
		if hr != nil {
			for _, l := range hr.Layers {
				result.ConfigChain = append(result.ConfigChain, engine.ConfigLayerStatus{
					Level:  string(l.Level),
					Path:   l.Path,
					Loaded: l.Loaded,
				})
			}
		}

		fmt.Printf("agent-sync %s\n", result.Version)
		fmt.Printf("  spec version:  %d\n", result.SpecVersion)

		// Show config chain if hierarchical loading was used.
		if len(result.ConfigChain) > 1 {
			fmt.Println("  config chain:")
			for _, layer := range result.ConfigChain {
				status := "not found"
				if layer.Loaded {
					status = "loaded"
				}
				fmt.Printf("    %-10s %s (%s)\n", layer.Level+":", layer.Path, status)
			}
		} else {
			fmt.Printf("  config:        %s\n", result.ConfigPath)
		}

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
