package lock

// Lockfile represents the agent-sync.lock file.
// See spec Section 4.
type Lockfile struct {
	Sources []LockedSource `yaml:"sources"`
	Version int            `yaml:"version"`
}

// LockedSource records the fully resolved, immutable state of a source.
// See spec Section 4.2.
type LockedSource struct {
	Name     string        `yaml:"name"`
	Type     string        `yaml:"type"`
	Repo     string        `yaml:"repo,omitempty"`
	Resolved ResolvedState `yaml:"resolved"`
	Status   string        `yaml:"status"`
}

// ResolvedState holds the resolved metadata for a source.
// Fields are populated based on source type.
type ResolvedState struct {
	// All source types: per-file content hashes.
	Files map[string]FileHash `yaml:"files,omitempty"`

	// Git source fields.
	Commit string `yaml:"commit,omitempty"`
	Tree   string `yaml:"tree,omitempty"`

	// URL source fields.
	URL    string `yaml:"url,omitempty"`
	SHA256 string `yaml:"sha256,omitempty"`

	// Local source fields.
	Path string `yaml:"path,omitempty"`
}

// FileHash records the content hash of a single file.
type FileHash struct {
	SHA256 string `yaml:"sha256"`
}
