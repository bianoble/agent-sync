package transform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/config"
)

// OverrideProcessor applies overrides to synced files.
type OverrideProcessor struct {
	ProjectRoot string
}

// ValidateOverrides checks that all override files exist at config validation time.
// Per spec Section 6.2 rule 5.
func (o *OverrideProcessor) ValidateOverrides(overrides []config.Override) error {
	for _, ov := range overrides {
		absPath := filepath.Join(o.ProjectRoot, ov.File)
		if _, err := os.Stat(absPath); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("override for '%s': file '%s' does not exist — create it or remove the override", ov.Target, ov.File)
		} else if err != nil {
			return fmt.Errorf("override for '%s': checking file '%s': %w", ov.Target, ov.File, err)
		}
	}
	return nil
}

// Apply applies all overrides to the synced file map.
// syncedFiles maps destination filename to content.
// Returns the modified file map.
func (o *OverrideProcessor) Apply(syncedFiles map[string][]byte, overrides []config.Override) (map[string][]byte, error) {
	result := make(map[string][]byte, len(syncedFiles))
	for k, v := range syncedFiles {
		content := make([]byte, len(v))
		copy(content, v)
		result[k] = content
	}

	for _, ov := range overrides {
		existing, exists := result[ov.Target]
		if !exists {
			return nil, fmt.Errorf("override for '%s': target file does not exist after sync — check that the source produces this file", ov.Target)
		}

		overrideContent, err := os.ReadFile(filepath.Join(o.ProjectRoot, ov.File))
		if err != nil {
			return nil, fmt.Errorf("override for '%s': reading override file '%s': %w", ov.Target, ov.File, err)
		}

		var merged []byte
		switch ov.Strategy {
		case "append":
			merged = appendContent(existing, overrideContent)
		case "prepend":
			merged = prependContent(existing, overrideContent)
		case "replace":
			merged = overrideContent
		default:
			return nil, fmt.Errorf("override for '%s': invalid strategy '%s'", ov.Target, ov.Strategy)
		}

		result[ov.Target] = merged
	}

	return result, nil
}

// ApplySingle applies a single override to file content.
func (o *OverrideProcessor) ApplySingle(content []byte, ov config.Override) ([]byte, error) {
	overrideContent, err := os.ReadFile(filepath.Join(o.ProjectRoot, ov.File))
	if err != nil {
		return nil, fmt.Errorf("reading override file '%s': %w", ov.File, err)
	}

	switch ov.Strategy {
	case "append":
		return appendContent(content, overrideContent), nil
	case "prepend":
		return prependContent(content, overrideContent), nil
	case "replace":
		return overrideContent, nil
	default:
		return nil, fmt.Errorf("invalid strategy '%s'", ov.Strategy)
	}
}

func appendContent(original, addition []byte) []byte {
	if len(original) > 0 && original[len(original)-1] != '\n' {
		original = append(original, '\n')
	}
	return append(original, addition...)
}

func prependContent(original, addition []byte) []byte {
	if len(addition) > 0 && addition[len(addition)-1] != '\n' {
		addition = append(addition, '\n')
	}
	return append(addition, original...)
}

// DetectConflicts checks if multiple sources write to the same destination
// without an explicit override. Different tool targets are allowed since
// they resolve to different paths.
func DetectConflicts(destinations map[string][]string, overrides []config.Override) error {
	overrideTargets := make(map[string]bool)
	for _, ov := range overrides {
		overrideTargets[ov.Target] = true
	}

	for dest, sources := range destinations {
		if len(sources) > 1 && !overrideTargets[filepath.Base(dest)] {
			return fmt.Errorf("conflict: multiple sources target '%s' (%s) — add an override or use different tools", dest, joinSources(sources))
		}
	}
	return nil
}

func joinSources(sources []string) string {
	if len(sources) <= 2 {
		result := ""
		for i, s := range sources {
			if i > 0 {
				result += ", "
			}
			result += s
		}
		return result
	}
	result := ""
	for i, s := range sources {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
