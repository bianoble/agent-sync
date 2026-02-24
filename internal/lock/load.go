package lock

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and validates an agent-sync.lock file.
func Load(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading lockfile %s: %w", path, err)
	}

	var lf Lockfile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing lockfile %s: %w", path, err)
	}

	if errs := Validate(&lf); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	return &lf, nil
}

// Save writes a lockfile atomically using a temp file and rename.
func Save(path string, lf *Lockfile) error {
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("marshaling lockfile: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing temp lockfile %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming temp lockfile to %s: %w", path, err)
	}

	return nil
}

// ValidationError holds multiple validation failures.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("lockfile validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

// Validate checks a Lockfile for semantic correctness.
// Returns a list of validation error messages (empty if valid).
func Validate(lf *Lockfile) []string {
	var errs []string

	// Version (Section 14).
	if lf.Version != 1 {
		errs = append(errs, fmt.Sprintf("unsupported version %d — only version 1 is supported", lf.Version))
	}

	// Check for duplicate source names (Section 8.3).
	names := make(map[string]bool)
	for i, src := range lf.Sources {
		prefix := fmt.Sprintf("locked_source[%d]", i)
		if src.Name != "" {
			prefix = fmt.Sprintf("locked source '%s'", src.Name)
		}

		if src.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: 'name' is required", prefix))
		} else if names[src.Name] {
			errs = append(errs, fmt.Sprintf("%s: duplicate source name '%s'", prefix, src.Name))
		} else {
			names[src.Name] = true
		}

		if src.Type == "" {
			errs = append(errs, fmt.Sprintf("%s: 'type' is required", prefix))
		}

		if src.Status == "" {
			errs = append(errs, fmt.Sprintf("%s: 'status' is required", prefix))
		}
	}

	return errs
}
