package config

import "testing"

func TestMergeVersionBothZero(t *testing.T) {
	base := &Config{Version: 0}
	overlay := &Config{Version: 0}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Version != 0 {
		t.Errorf("version = %d, want 0", merged.Version)
	}
}

func TestMergeVersionOverlayZeroInheritsBase(t *testing.T) {
	base := &Config{Version: 1}
	overlay := &Config{Version: 0}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Version != 1 {
		t.Errorf("version = %d, want 1", merged.Version)
	}
}

func TestMergeVersionBaseZeroInheritsOverlay(t *testing.T) {
	base := &Config{Version: 0}
	overlay := &Config{Version: 1}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Version != 1 {
		t.Errorf("version = %d, want 1", merged.Version)
	}
}

func TestMergeVersionBothSame(t *testing.T) {
	base := &Config{Version: 1}
	overlay := &Config{Version: 1}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Version != 1 {
		t.Errorf("version = %d, want 1", merged.Version)
	}
}

func TestMergeNamedToolDefsEmptyBase(t *testing.T) {
	base := &Config{Version: 1}
	overlay := &Config{
		Version: 1,
		ToolDefinitions: []ToolDefinition{
			{Name: "custom", Destination: ".custom/"},
		},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(merged.ToolDefinitions) != 1 {
		t.Errorf("tool_definitions = %d, want 1", len(merged.ToolDefinitions))
	}
}

func TestMergeNamedToolDefsEmptyOverlay(t *testing.T) {
	base := &Config{
		Version: 1,
		ToolDefinitions: []ToolDefinition{
			{Name: "custom", Destination: ".custom/"},
		},
	}
	overlay := &Config{Version: 1}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(merged.ToolDefinitions) != 1 {
		t.Errorf("tool_definitions = %d, want 1", len(merged.ToolDefinitions))
	}
}

func TestMergeSourcesEmptyBase(t *testing.T) {
	base := &Config{Version: 1}
	overlay := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
	}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(merged.Sources) != 1 {
		t.Errorf("sources = %d, want 1", len(merged.Sources))
	}
}

func TestMergeSourcesEmptyOverlay(t *testing.T) {
	base := &Config{
		Version: 1,
		Sources: []Source{{Name: "s", Type: "local", Path: "./a/"}},
	}
	overlay := &Config{Version: 1}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(merged.Sources) != 1 {
		t.Errorf("sources = %d, want 1", len(merged.Sources))
	}
}

func TestMergeVariablesBothEmpty(t *testing.T) {
	base := &Config{Version: 1}
	overlay := &Config{Version: 1}

	merged, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Variables != nil {
		t.Errorf("variables should be nil, got %v", merged.Variables)
	}
}

func TestMergeAllSingle(t *testing.T) {
	cfg := &Config{Version: 1, Variables: map[string]string{"k": "v"}}
	merged, err := MergeAll([]*Config{cfg})
	if err != nil {
		t.Fatalf("MergeAll: %v", err)
	}
	if merged.Version != 1 {
		t.Errorf("version = %d, want 1", merged.Version)
	}
	if merged.Variables["k"] != "v" {
		t.Errorf("variables[k] = %q, want v", merged.Variables["k"])
	}
}
