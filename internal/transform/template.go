package transform

import (
	"bytes"
	"fmt"
	"text/template"
	"unicode/utf8"
)

// TemplateTransform applies Go text/template variable substitution.
type TemplateTransform struct{}

// Apply processes a single file's content through template substitution.
// vars is the merged variable map (global variables + per-transform vars).
func (t *TemplateTransform) Apply(content []byte, vars map[string]string) ([]byte, error) {
	// Skip binary files.
	if !utf8.Valid(content) || containsNullByte(content) {
		return content, nil
	}

	tmpl, err := template.New("").Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// ApplyToFiles processes multiple files through template substitution.
func (t *TemplateTransform) ApplyToFiles(files []TransformFile, vars map[string]string) ([]TransformFile, error) {
	result := make([]TransformFile, len(files))
	for i, f := range files {
		out, err := t.Apply(f.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("file '%s': %w", f.RelPath, err)
		}
		result[i] = TransformFile{
			RelPath: f.RelPath,
			Content: out,
		}
	}
	return result, nil
}

// TransformFile holds a file path and its content for transformation.
type TransformFile struct {
	RelPath string
	Content []byte
}

// MergeVars merges global variables with per-transform variables.
// Per-transform vars override global vars.
func MergeVars(global map[string]string, perTransform map[string]string) map[string]string {
	merged := make(map[string]string, len(global)+len(perTransform))
	for k, v := range global {
		merged[k] = v
	}
	for k, v := range perTransform {
		merged[k] = v
	}
	return merged
}

func containsNullByte(data []byte) bool {
	return bytes.ContainsRune(data, 0)
}
