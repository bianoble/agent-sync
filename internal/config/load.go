package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and validates an agent-sync.yaml configuration file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if errs := Validate(&cfg); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	return &cfg, nil
}

// ValidationError holds multiple validation failures.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

// Validate checks a Config for semantic correctness.
// Returns a list of validation error messages (empty if valid).
func Validate(cfg *Config) []string {
	var errs []string

	// Version (Section 14).
	if cfg.Version != 1 {
		errs = append(errs, fmt.Sprintf("unsupported version %d — only version 1 is supported", cfg.Version))
	}

	// Sources.
	if len(cfg.Sources) == 0 {
		errs = append(errs, "at least one source is required")
	}

	sourceNames := make(map[string]bool)
	for i, src := range cfg.Sources {
		prefix := fmt.Sprintf("source[%d]", i)
		if src.Name != "" {
			prefix = fmt.Sprintf("source '%s'", src.Name)
		}

		if src.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: 'name' is required", prefix))
		} else if sourceNames[src.Name] {
			errs = append(errs, fmt.Sprintf("%s: duplicate source name '%s'", prefix, src.Name))
		} else {
			sourceNames[src.Name] = true
		}

		errs = append(errs, validateSource(src, prefix)...)
	}

	// Targets (Section 7.3).
	for i, tgt := range cfg.Targets {
		prefix := fmt.Sprintf("target[%d]", i)
		if tgt.Source != "" {
			prefix = fmt.Sprintf("target for source '%s'", tgt.Source)
		}

		if tgt.Source == "" {
			errs = append(errs, fmt.Sprintf("%s: 'source' is required", prefix))
		} else if !sourceNames[tgt.Source] {
			errs = append(errs, fmt.Sprintf("%s: references undefined source '%s'", prefix, tgt.Source))
		}

		if len(tgt.Tools) > 0 && tgt.Destination != "" {
			errs = append(errs, fmt.Sprintf("%s: 'tools' and 'destination' are mutually exclusive — use one or the other", prefix))
		}
		if len(tgt.Tools) == 0 && tgt.Destination == "" {
			errs = append(errs, fmt.Sprintf("%s: one of 'tools' or 'destination' is required", prefix))
		}
	}

	// Overrides (Section 6.2).
	for i, ov := range cfg.Overrides {
		prefix := fmt.Sprintf("override[%d]", i)
		if ov.Target != "" {
			prefix = fmt.Sprintf("override for '%s'", ov.Target)
		}

		if ov.Target == "" {
			errs = append(errs, fmt.Sprintf("%s: 'target' is required", prefix))
		}
		if ov.File == "" {
			errs = append(errs, fmt.Sprintf("%s: 'file' is required", prefix))
		}

		switch ov.Strategy {
		case "append", "prepend", "replace":
			// valid
		case "":
			errs = append(errs, fmt.Sprintf("%s: 'strategy' is required — must be one of: append, prepend, replace", prefix))
		default:
			errs = append(errs, fmt.Sprintf("%s: invalid strategy '%s' — must be one of: append, prepend, replace", prefix, ov.Strategy))
		}
	}

	// Transforms.
	for i, tx := range cfg.Transforms {
		prefix := fmt.Sprintf("transform[%d]", i)
		if tx.Source != "" {
			prefix = fmt.Sprintf("transform for source '%s'", tx.Source)
		}

		if tx.Source == "" {
			errs = append(errs, fmt.Sprintf("%s: 'source' is required", prefix))
		} else if !sourceNames[tx.Source] {
			errs = append(errs, fmt.Sprintf("%s: references undefined source '%s'", prefix, tx.Source))
		}

		switch tx.Type {
		case "template":
			// vars is optional
		case "custom":
			if tx.Command == "" {
				errs = append(errs, fmt.Sprintf("%s: custom transform requires 'command'", prefix))
			}
		case "":
			errs = append(errs, fmt.Sprintf("%s: 'type' is required — must be one of: template, custom", prefix))
		default:
			errs = append(errs, fmt.Sprintf("%s: invalid type '%s' — must be one of: template, custom", prefix, tx.Type))
		}
	}

	// Tool definitions.
	for i, td := range cfg.ToolDefinitions {
		prefix := fmt.Sprintf("tool_definition[%d]", i)
		if td.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: 'name' is required", prefix))
		}
		if td.Destination == "" {
			errs = append(errs, fmt.Sprintf("%s: 'destination' is required", prefix))
		}
	}

	return errs
}

func validateSource(src Source, prefix string) []string {
	var errs []string

	switch src.Type {
	case "git":
		if src.Repo == "" {
			errs = append(errs, fmt.Sprintf("%s: type 'git' requires 'repo' — add 'repo: https://...' to the source definition", prefix))
		}
		if src.Ref == "" {
			errs = append(errs, fmt.Sprintf("%s: type 'git' requires 'ref' — add 'ref: <tag-or-branch>' to the source definition", prefix))
		}
	case "url":
		if src.URL == "" {
			errs = append(errs, fmt.Sprintf("%s: type 'url' requires 'url' — add 'url: https://...' to the source definition", prefix))
		}
		if src.Checksum == "" {
			errs = append(errs, fmt.Sprintf("%s: type 'url' requires 'checksum' — add 'checksum: sha256:<hex>' to the source definition", prefix))
		}
	case "local":
		if src.Path == "" {
			errs = append(errs, fmt.Sprintf("%s: type 'local' requires 'path' — add 'path: ./relative/path/' to the source definition", prefix))
		}
	case "":
		errs = append(errs, fmt.Sprintf("%s: 'type' is required — must be one of: git, url, local", prefix))
	default:
		errs = append(errs, fmt.Sprintf("%s: unknown source type '%s' — must be one of: git, url, local", prefix, src.Type))
	}

	return errs
}
