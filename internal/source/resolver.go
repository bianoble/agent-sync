package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bianoble/agent-sync/internal/config"
)

// Resolver resolves a source definition into its immutable state and fetches files.
type Resolver interface {
	// Resolve resolves a source to its current upstream state.
	Resolve(ctx context.Context, src config.Source, projectRoot string) (*ResolvedSource, error)

	// Fetch retrieves the actual file content for a resolved source.
	Fetch(ctx context.Context, resolved *ResolvedSource) ([]FetchedFile, error)
}

// ResolvedSource holds the fully resolved, immutable state of a source.
type ResolvedSource struct {
	Files  map[string]string // relative path -> sha256 hash
	Name   string
	Type   string
	Commit string // git only
	Tree   string // git only
	URL    string // url only
	Repo   string // git only
	Path   string // local only
}

// FetchedFile holds the content of a single fetched file.
type FetchedFile struct {
	RelPath string
	SHA256  string
	Content []byte
}

// SourceError represents an error associated with a specific source operation.
type SourceError struct {
	Source    string
	Operation string
	Err       error
	Hint      string
}

func (e *SourceError) Error() string {
	msg := fmt.Sprintf("%s: %s failed: %s", e.Source, e.Operation, e.Err)
	if e.Hint != "" {
		msg += " — " + e.Hint
	}
	return msg
}

func (e *SourceError) Unwrap() error {
	return e.Err
}

// Registry maps source type strings to Resolver implementations.
type Registry struct {
	resolvers map[string]Resolver
}

// NewRegistry creates a new empty source resolver registry.
func NewRegistry() *Registry {
	return &Registry{resolvers: make(map[string]Resolver)}
}

// Register adds a resolver for the given source type.
func (r *Registry) Register(sourceType string, resolver Resolver) {
	r.resolvers[sourceType] = resolver
}

// Get returns the resolver for the given source type.
func (r *Registry) Get(sourceType string) (Resolver, error) {
	res, ok := r.resolvers[sourceType]
	if !ok {
		return nil, fmt.Errorf("unknown source type '%s' — supported types: %s", sourceType, r.supportedTypes())
	}
	return res, nil
}

func (r *Registry) supportedTypes() string {
	types := make([]string, 0, len(r.resolvers))
	for t := range r.resolvers {
		types = append(types, t)
	}
	if len(types) == 0 {
		return "(none registered)"
	}
	return fmt.Sprintf("%v", types)
}

// FS abstracts filesystem operations for testing and embedding.
type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Walk(root string, fn filepath.WalkFunc) error
	Stat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	Remove(path string) error
	Rename(oldpath, newpath string) error
}

// HTTPClient abstracts HTTP operations for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// OSFS implements FS using the real operating system filesystem.
type OSFS struct{}

func (OSFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (OSFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (OSFS) Walk(root string, fn filepath.WalkFunc) error { return filepath.Walk(root, fn) }
func (OSFS) Stat(path string) (os.FileInfo, error)        { return os.Stat(path) }
func (OSFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (OSFS) Remove(path string) error                     { return os.Remove(path) }
func (OSFS) Rename(oldpath, newpath string) error         { return os.Rename(oldpath, newpath) }

// DefaultHTTPClient returns an HTTPClient using http.DefaultClient.
type DefaultHTTPClient struct{}

func (DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// Discard is an io.Writer that discards all data (re-exported for convenience).
var Discard io.Writer = io.Discard
