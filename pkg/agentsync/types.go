package agentsync

// FileAction represents an action taken on a single file during sync or prune.
type FileAction struct {
	Path   string
	Action string // "written", "skipped", "removed"
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
// See spec Section 10.2.
type SyncResult struct {
	Written []FileAction
	Skipped []FileAction
	Errors  []SourceError
}

// CheckResult holds the outcome of a check operation.
// See spec Section 10.2.
type CheckResult struct {
	Clean   bool
	Drifted []DriftEntry
	Missing []string
}

// VerifyResult holds the outcome of a verify operation.
// See spec Section 10.2.
type VerifyResult struct {
	UpToDate []string
	Changed  []SourceDelta
	Errors   []SourceError
}

// SyncOptions configures a sync operation.
type SyncOptions struct {
	DryRun bool
}

// PruneOptions configures a prune operation.
type PruneOptions struct {
	DryRun bool
}

// PruneResult holds the outcome of a prune operation.
// See spec Section 10.2.
type PruneResult struct {
	Removed []FileAction
	Errors  []SourceError
}
