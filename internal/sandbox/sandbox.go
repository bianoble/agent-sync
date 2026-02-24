package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath checks if targetPath is safely within projectRoot.
// It resolves symlinks, normalizes paths, and verifies containment.
// Returns the resolved absolute path or an error.
func ValidatePath(projectRoot, targetPath string) (string, error) {
	// Resolve the project root to its real path.
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolving project root: %w", err)
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", fmt.Errorf("resolving project root symlinks: %w", err)
	}

	// Build the candidate path.
	candidate := filepath.Join(realRoot, targetPath)
	candidate = filepath.Clean(candidate)

	// Resolve symlinks in the candidate path.
	// The path may not exist yet, so resolve as much as we can.
	resolved, err := resolveExistingPath(candidate)
	if err != nil {
		return "", fmt.Errorf("resolving target path: %w", err)
	}

	// Ensure the resolved path is within the project root.
	// Add trailing separator to avoid prefix matching "projectroot2" for "projectroot".
	rootPrefix := realRoot + string(filepath.Separator)
	if resolved != realRoot && !strings.HasPrefix(resolved, rootPrefix) {
		return "", fmt.Errorf("path '%s' resolves to '%s' which is outside the project root '%s'", targetPath, resolved, realRoot)
	}

	return resolved, nil
}

// resolveExistingPath resolves symlinks for the longest existing prefix of the path,
// then appends the non-existing suffix. This handles paths that don't fully exist yet.
func resolveExistingPath(path string) (string, error) {
	// Try resolving the full path first.
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}

	// Walk up to find the longest existing prefix.
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	if dir == path {
		// We've reached the root without finding anything.
		return path, nil
	}

	resolvedDir, err := resolveExistingPath(dir)
	if err != nil {
		return "", err
	}

	return filepath.Join(resolvedDir, base), nil
}

// SafeWrite atomically writes content to a path within the project root.
func SafeWrite(projectRoot, relPath string, content []byte, perm os.FileMode) error {
	resolved, err := ValidatePath(projectRoot, relPath)
	if err != nil {
		return err
	}

	// Create parent directories.
	dir := filepath.Dir(resolved)
	parentResolved, err := ValidatePath(projectRoot, filepath.Dir(relPath))
	if err != nil {
		return fmt.Errorf("parent directory escapes sandbox: %w", err)
	}
	_ = parentResolved
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Write to temp file in the same directory (ensures same filesystem for rename).
	tmp, err := os.CreateTemp(dir, ".agent-sync-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on any failure.
	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, resolved); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", resolved, err)
	}

	success = true
	return nil
}

// SafeRemove removes a file within the project root sandbox.
func SafeRemove(projectRoot, relPath string) error {
	resolved, err := ValidatePath(projectRoot, relPath)
	if err != nil {
		return err
	}
	return os.Remove(resolved)
}

// SafeMkdirAll creates directories within the sandbox.
func SafeMkdirAll(projectRoot, relPath string, perm os.FileMode) error {
	resolved, err := ValidatePath(projectRoot, relPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(resolved, perm)
}
