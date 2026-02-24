package target

import (
	"strings"
	"testing"

	"github.com/bianoble/agent-sync/internal/config"
)

func TestBuiltinToolResolution(t *testing.T) {
	tm := NewToolMap(nil)

	tests := []struct {
		tool string
		want string
	}{
		{"cursor", ".cursor/rules/"},
		{"claude-code", ".claude/"},
		{"copilot", ".github/copilot/"},
		{"windsurf", ".windsurf/rules/"},
		{"cline", ".cline/rules/"},
		{"codex", ".codex/"},
	}

	for _, tt := range tests {
		got, err := tm.Resolve(tt.tool)
		if err != nil {
			t.Errorf("Resolve(%q): %v", tt.tool, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.tool, got, tt.want)
		}
	}
}

func TestCustomToolOverridesBuiltin(t *testing.T) {
	tm := NewToolMap([]config.ToolDefinition{
		{Name: "cursor", Destination: ".cursor/custom-rules/"},
	})

	got, err := tm.Resolve("cursor")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != ".cursor/custom-rules/" {
		t.Errorf("got %q, want %q", got, ".cursor/custom-rules/")
	}
}

func TestCustomToolNew(t *testing.T) {
	tm := NewToolMap([]config.ToolDefinition{
		{Name: "internal-agent", Destination: ".internal/agent-config/"},
	})

	got, err := tm.Resolve("internal-agent")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != ".internal/agent-config/" {
		t.Errorf("got %q, want %q", got, ".internal/agent-config/")
	}
}

func TestUnknownToolError(t *testing.T) {
	tm := NewToolMap(nil)
	_, err := tm.Resolve("mytool")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool 'mytool'") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "tool_definitions") {
		t.Errorf("error should suggest tool_definitions: %v", err)
	}
}

func TestResolveTargetWithTools(t *testing.T) {
	tm := NewToolMap(nil)
	tgt := config.Target{Source: "rules", Tools: []string{"cursor", "claude-code"}}

	results, err := tm.ResolveTarget(tgt)
	if err != nil {
		t.Fatalf("ResolveTarget: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Destination != ".cursor/rules/" {
		t.Errorf("results[0].destination = %q", results[0].Destination)
	}
	if results[0].ToolName != "cursor" {
		t.Errorf("results[0].tool = %q", results[0].ToolName)
	}
	if results[1].Destination != ".claude/" {
		t.Errorf("results[1].destination = %q", results[1].Destination)
	}
}

func TestResolveTargetWithDestination(t *testing.T) {
	tm := NewToolMap(nil)
	tgt := config.Target{Source: "rules", Destination: ".custom/rules/"}

	results, err := tm.ResolveTarget(tgt)
	if err != nil {
		t.Fatalf("ResolveTarget: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Destination != ".custom/rules/" {
		t.Errorf("destination = %q", results[0].Destination)
	}
	if results[0].ToolName != "" {
		t.Errorf("tool should be empty for explicit destination, got %q", results[0].ToolName)
	}
}

func TestResolveTargetMutualExclusive(t *testing.T) {
	tm := NewToolMap(nil)
	tgt := config.Target{Source: "rules", Tools: []string{"cursor"}, Destination: ".out/"}

	_, err := tm.ResolveTarget(tgt)
	if err == nil {
		t.Fatal("expected error for mutual exclusion")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKnownToolsIncludesCustom(t *testing.T) {
	tm := NewToolMap([]config.ToolDefinition{
		{Name: "custom-tool", Destination: ".custom/"},
	})

	tools := tm.KnownTools()
	found := false
	for _, tool := range tools {
		if tool == "custom-tool" {
			found = true
		}
	}
	if !found {
		t.Errorf("KnownTools should include custom-tool, got: %v", tools)
	}
	// Should also include builtins.
	if len(tools) < 7 {
		t.Errorf("KnownTools should have at least 7 entries (6 builtin + 1 custom), got %d", len(tools))
	}
}

func TestIsCustom(t *testing.T) {
	tm := NewToolMap([]config.ToolDefinition{
		{Name: "custom-tool", Destination: ".custom/"},
		{Name: "cursor", Destination: ".cursor/override/"},
	})

	if tm.IsCustom("custom-tool") != true {
		t.Error("custom-tool should be custom")
	}
	if tm.IsCustom("cursor") != false {
		t.Error("cursor is a built-in (even if overridden)")
	}
	if tm.IsCustom("copilot") != false {
		t.Error("copilot should not be custom")
	}
}
