package transform

import (
	"strings"
	"testing"
)

func TestTemplateSimpleSubstitution(t *testing.T) {
	tx := &TemplateTransform{}
	content := []byte("Project: {{ .project }}")
	vars := map[string]string{"project": "my-project"}

	result, err := tx.Apply(content, vars)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if string(result) != "Project: my-project" {
		t.Errorf("got %q", string(result))
	}
}

func TestTemplateMultipleVars(t *testing.T) {
	tx := &TemplateTransform{}
	content := []byte("{{ .project }} uses {{ .language }}")
	vars := map[string]string{"project": "test", "language": "go"}

	result, err := tx.Apply(content, vars)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if string(result) != "test uses go" {
		t.Errorf("got %q", string(result))
	}
}

func TestTemplateMissingVarError(t *testing.T) {
	tx := &TemplateTransform{}
	content := []byte("{{ .missing }}")
	vars := map[string]string{"project": "test"}

	_, err := tx.Apply(content, vars)
	if err == nil {
		t.Fatal("expected error for missing variable")
	}
}

func TestTemplateSkipsBinaryContent(t *testing.T) {
	tx := &TemplateTransform{}
	binary := []byte{0x00, 0x01, 0x02, 0x03}
	vars := map[string]string{"key": "value"}

	result, err := tx.Apply(binary, vars)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Should return binary content unchanged.
	if len(result) != len(binary) {
		t.Errorf("binary content should be unchanged, got len %d", len(result))
	}
}

func TestTemplateNoVarsPassthrough(t *testing.T) {
	tx := &TemplateTransform{}
	content := []byte("No template syntax here.")
	vars := map[string]string{}

	result, err := tx.Apply(content, vars)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if string(result) != "No template syntax here." {
		t.Errorf("got %q", string(result))
	}
}

func TestTemplateInvalidSyntaxError(t *testing.T) {
	tx := &TemplateTransform{}
	content := []byte("{{ .unclosed")
	vars := map[string]string{}

	_, err := tx.Apply(content, vars)
	if err == nil {
		t.Fatal("expected error for invalid template syntax")
	}
	if !strings.Contains(err.Error(), "parsing template") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyToFiles(t *testing.T) {
	tx := &TemplateTransform{}
	files := []TransformFile{
		{RelPath: "a.md", Content: []byte("Hello {{ .name }}")},
		{RelPath: "b.md", Content: []byte("Bye {{ .name }}")},
	}
	vars := map[string]string{"name": "world"}

	result, err := tx.ApplyToFiles(files, vars)
	if err != nil {
		t.Fatalf("ApplyToFiles: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d files, want 2", len(result))
	}
	if string(result[0].Content) != "Hello world" {
		t.Errorf("file[0] = %q", string(result[0].Content))
	}
	if string(result[1].Content) != "Bye world" {
		t.Errorf("file[1] = %q", string(result[1].Content))
	}
}

func TestMergeVars(t *testing.T) {
	global := map[string]string{"a": "1", "b": "2"}
	local := map[string]string{"b": "override", "c": "3"}

	merged := MergeVars(global, local)

	if merged["a"] != "1" {
		t.Errorf("a = %q, want 1", merged["a"])
	}
	if merged["b"] != "override" {
		t.Errorf("b = %q, want override", merged["b"])
	}
	if merged["c"] != "3" {
		t.Errorf("c = %q, want 3", merged["c"])
	}
}

func TestDeterminism(t *testing.T) {
	tx := &TemplateTransform{}
	content := []byte("{{ .a }} {{ .b }} {{ .c }}")
	vars := map[string]string{"a": "x", "b": "y", "c": "z"}

	first, err := tx.Apply(content, vars)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		result, err := tx.Apply(content, vars)
		if err != nil {
			t.Fatal(err)
		}
		if string(result) != string(first) {
			t.Fatalf("iteration %d: non-deterministic output %q vs %q", i, string(result), string(first))
		}
	}
}
