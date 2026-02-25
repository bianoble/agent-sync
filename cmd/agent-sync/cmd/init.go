package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initForce bool

// initTemplate is the default agent-sync.yaml scaffold.
// It includes a working git source example and commented-out alternatives.
const initTemplate = `# agent-sync configuration
# Docs: https://github.com/bianoble/agent-sync
version: 1

sources:
  # Git repository (most common)
  - name: team-rules
    type: git
    repo: https://github.com/your-org/agent-rules.git
    ref: main
    # paths:                  # optional: sync only these paths
    #   - rules/

  # Single file from a URL
  # - name: security-policy
  #   type: url
  #   url: https://example.com/security.md
  #   checksum: sha256:abc123...

  # Local directory
  # - name: local-rules
  #   type: local
  #   path: ./agents/rules/

targets:
  - source: team-rules
    tools: [cursor, claude-code]
    # Or use an explicit destination:
    # destination: .cursor/rules/

  # Built-in tools: cursor, claude-code, copilot, windsurf, cline, codex

# variables:
#   org: your-org
#   env: production

# transforms:
#   - source: team-rules
#     type: template
#     vars:
#       team: my-team

# overrides:
#   - target: .cursor/rules/intro.md
#     strategy: prepend
#     file: ./local/header.md

# tool_definitions:
#   - name: my-custom-tool
#     destination: .my-tool/rules/
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter agent-sync.yaml configuration",
	Long: `Creates an agent-sync.yaml file in the current directory with a well-commented
template including a git source example and documented alternatives for URL and
local sources.

Use --force to overwrite an existing configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outPath := configPath
		if !filepath.IsAbs(outPath) {
			abs, err := filepath.Abs(outPath)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}
			outPath = abs
		}

		if !initForce {
			if _, err := os.Stat(outPath); err == nil {
				return fmt.Errorf("%s already exists (use --force to overwrite)", outPath)
			}
		}

		if err := os.WriteFile(outPath, []byte(initTemplate), 0644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		info("Created %s", outPath)
		info("")
		info("Next steps:")
		info("  1. Edit the file to point at your sources")
		info("  2. Run 'agent-sync update' to resolve and lock")
		info("  3. Run 'agent-sync sync' to write files to targets")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
	rootCmd.AddCommand(initCmd)
}
