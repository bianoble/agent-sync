package target

import (
	"fmt"

	"github.com/bianoble/agent-sync/internal/config"
)

// builtinTools defines the default tool path mappings per spec Section 3.2.
var builtinTools = map[string]string{
	"cursor":     ".cursor/rules/",
	"claude-code": ".claude/",
	"copilot":    ".github/copilot/",
	"windsurf":   ".windsurf/rules/",
	"cline":      ".cline/rules/",
	"codex":      ".codex/",
}

// ToolMap resolves tool names to destination paths.
type ToolMap struct {
	definitions map[string]string
}

// NewToolMap creates a ToolMap with built-in definitions and optional custom overrides.
func NewToolMap(customDefs []config.ToolDefinition) *ToolMap {
	defs := make(map[string]string, len(builtinTools)+len(customDefs))
	for name, dest := range builtinTools {
		defs[name] = dest
	}
	for _, td := range customDefs {
		defs[td.Name] = td.Destination
	}
	return &ToolMap{definitions: defs}
}

// Resolve returns the destination path for a tool name.
func (tm *ToolMap) Resolve(toolName string) (string, error) {
	dest, ok := tm.definitions[toolName]
	if !ok {
		return "", fmt.Errorf("unknown tool '%s' — define it in tool_definitions: [{name: %s, destination: .%s/}]", toolName, toolName, toolName)
	}
	return dest, nil
}

// ResolvedTarget represents a source mapped to a specific destination.
type ResolvedTarget struct {
	Source      string
	Destination string
	ToolName    string // empty for explicit destination targets
}

// ResolveTarget resolves a target entry to one or more destination paths.
// Returns one ResolvedTarget per tool, or a single one for explicit destinations.
func (tm *ToolMap) ResolveTarget(tgt config.Target) ([]ResolvedTarget, error) {
	if len(tgt.Tools) > 0 && tgt.Destination != "" {
		return nil, fmt.Errorf("target for source '%s': 'tools' and 'destination' are mutually exclusive — use one or the other", tgt.Source)
	}

	if tgt.Destination != "" {
		return []ResolvedTarget{
			{Source: tgt.Source, Destination: tgt.Destination},
		}, nil
	}

	results := make([]ResolvedTarget, 0, len(tgt.Tools))
	for _, tool := range tgt.Tools {
		dest, err := tm.Resolve(tool)
		if err != nil {
			return nil, err
		}
		results = append(results, ResolvedTarget{
			Source:      tgt.Source,
			Destination: dest,
			ToolName:    tool,
		})
	}
	return results, nil
}

// KnownTools returns all known tool names (built-in + custom).
func (tm *ToolMap) KnownTools() []string {
	names := make([]string, 0, len(tm.definitions))
	for name := range tm.definitions {
		names = append(names, name)
	}
	return names
}

// IsCustom returns whether a tool name is a custom definition (not built-in).
func (tm *ToolMap) IsCustom(toolName string) bool {
	_, isBuiltin := builtinTools[toolName]
	_, isDefined := tm.definitions[toolName]
	return isDefined && !isBuiltin
}
