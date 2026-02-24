package engine

// FileAction represents an action taken on a single file during sync or prune.
type FileAction struct {
	Path   string
	Action string // "written", "modified", "new", "skipped", "removed", "unchanged"
}

// SourceError represents an error associated with a specific source.
type SourceError struct {
	Source string
	Err    error
}

func (e SourceError) Error() string {
	return e.Source + ": " + e.Err.Error()
}

func (e SourceError) Unwrap() error {
	return e.Err
}

// DriftEntry represents a file that has drifted from the expected state.
type DriftEntry struct {
	Path     string
	Expected string
	Actual   string
}

// SourceDelta represents a change detected in an upstream source.
type SourceDelta struct {
	Source string
	Before string
	After  string
}

// SyncResult holds the outcome of a sync operation.
type SyncResult struct {
	Written []FileAction
	Skipped []FileAction
	Errors  []SourceError
}

// CheckResult holds the outcome of a check operation.
type CheckResult struct {
	Clean   bool
	Drifted []DriftEntry
	Missing []string
}

// VerifyResult holds the outcome of a verify operation.
type VerifyResult struct {
	UpToDate []string
	Changed  []SourceDelta
	Errors   []SourceError
}

// PruneResult holds the outcome of a prune operation.
type PruneResult struct {
	Removed []FileAction
	Errors  []SourceError
}
