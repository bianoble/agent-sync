package config

import "fmt"

// Merge combines two configs where overlay takes precedence over base.
// This implements the hierarchical merge semantics:
//   - version: must agree if both declare it (non-zero); fatal error on mismatch
//   - variables: deep merge, overlay keys win
//   - sources: merge by name — same name in overlay replaces base entry entirely
//   - tool_definitions: merge by name — same name in overlay replaces base entry
//   - targets, overrides, transforms: concatenate (base first, then overlay)
func Merge(base, overlay *Config) (*Config, error) {
	if base == nil {
		return overlay, nil
	}
	if overlay == nil {
		return base, nil
	}

	result := &Config{}

	// Version: must agree if both are non-zero.
	if err := mergeVersion(base.Version, overlay.Version, &result.Version); err != nil {
		return nil, err
	}

	// Variables: deep merge with overlay winning.
	result.Variables = mergeVariables(base.Variables, overlay.Variables)

	// Sources: merge by name.
	result.Sources = mergeNamedSources(base.Sources, overlay.Sources)

	// ToolDefinitions: merge by name.
	result.ToolDefinitions = mergeNamedToolDefs(base.ToolDefinitions, overlay.ToolDefinitions)

	// Targets: concatenate.
	result.Targets = append(result.Targets, base.Targets...)
	result.Targets = append(result.Targets, overlay.Targets...)

	// Overrides: concatenate.
	result.Overrides = append(result.Overrides, base.Overrides...)
	result.Overrides = append(result.Overrides, overlay.Overrides...)

	// Transforms: concatenate.
	result.Transforms = append(result.Transforms, base.Transforms...)
	result.Transforms = append(result.Transforms, overlay.Transforms...)

	return result, nil
}

// MergeAll merges multiple configs in order (lowest precedence first).
// Returns an error if any version mismatch is found.
func MergeAll(configs []*Config) (*Config, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configs to merge")
	}

	result := configs[0]
	for i := 1; i < len(configs); i++ {
		var err error
		result, err = Merge(result, configs[i])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func mergeVersion(base, overlay int, out *int) error {
	switch {
	case base == 0 && overlay == 0:
		*out = 0 // neither declares; validation will catch this
	case base == 0:
		*out = overlay
	case overlay == 0:
		*out = base
	case base == overlay:
		*out = base
	default:
		return fmt.Errorf("config version mismatch: one layer declares version %d, another declares version %d — all config layers must agree on version", base, overlay)
	}
	return nil
}

func mergeVariables(base, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}

	result := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		result[k] = v // overlay wins
	}
	return result
}

func mergeNamedSources(base, overlay []Source) []Source {
	if len(base) == 0 {
		return overlay
	}
	if len(overlay) == 0 {
		return base
	}

	// Build index of overlay names for quick lookup.
	overlayNames := make(map[string]bool, len(overlay))
	for _, s := range overlay {
		overlayNames[s.Name] = true
	}

	// Start with base entries that aren't overridden.
	var result []Source
	for _, s := range base {
		if !overlayNames[s.Name] {
			result = append(result, s)
		}
	}

	// Append all overlay entries (includes replacements and new ones).
	result = append(result, overlay...)

	return result
}

func mergeNamedToolDefs(base, overlay []ToolDefinition) []ToolDefinition {
	if len(base) == 0 {
		return overlay
	}
	if len(overlay) == 0 {
		return base
	}

	overlayNames := make(map[string]bool, len(overlay))
	for _, td := range overlay {
		overlayNames[td.Name] = true
	}

	var result []ToolDefinition
	for _, td := range base {
		if !overlayNames[td.Name] {
			result = append(result, td)
		}
	}

	result = append(result, overlay...)

	return result
}
