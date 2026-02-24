package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bianoble/agent-sync/internal/config"
)

// URLResolver resolves and fetches files from HTTP(S) URLs.
type URLResolver struct {
	Client  HTTPClient
	MaxSize int64         // max file size in bytes (0 = no limit)
	Timeout time.Duration // fetch timeout (0 = no extra timeout beyond context)
}

func (u *URLResolver) Resolve(ctx context.Context, src config.Source, projectRoot string) (*ResolvedSource, error) {
	if src.URL == "" {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("url is required")}
	}
	if src.Checksum == "" {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("checksum is required"), Hint: "add 'checksum: sha256:<hex>'"}
	}

	algo, expectedHash, err := parseChecksum(src.Checksum)
	if err != nil {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: err}
	}
	if algo != "sha256" {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("unsupported checksum algorithm '%s' — only 'sha256' is supported", algo)}
	}

	content, err := u.fetchURL(ctx, src.URL, src.Name)
	if err != nil {
		return nil, err
	}

	actualHash := computeURLHash(content)
	if actualHash != expectedHash {
		return nil, &SourceError{
			Source:    src.Name,
			Operation: "resolve",
			Err:      fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash),
			Hint:     "the upstream content has changed — update the checksum in your config",
		}
	}

	fileName := path.Base(src.URL)
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "file"
	}

	return &ResolvedSource{
		Name:  src.Name,
		Type:  "url",
		URL:   src.URL,
		Files: map[string]string{fileName: actualHash},
	}, nil
}

func (u *URLResolver) Fetch(ctx context.Context, resolved *ResolvedSource) ([]FetchedFile, error) {
	if resolved.URL == "" {
		return nil, &SourceError{Source: resolved.Name, Operation: "fetch", Err: fmt.Errorf("resolved source missing URL")}
	}

	content, err := u.fetchURL(ctx, resolved.URL, resolved.Name)
	if err != nil {
		return nil, err
	}

	// Verify against resolved hashes.
	actualHash := computeURLHash(content)
	for relPath, expectedHash := range resolved.Files {
		if actualHash != expectedHash {
			return nil, &SourceError{
				Source:    resolved.Name,
				Operation: "fetch",
				Err:      fmt.Errorf("hash mismatch for %s: expected %s, got %s", relPath, expectedHash, actualHash),
			}
		}
		return []FetchedFile{
			{RelPath: relPath, Content: content, SHA256: actualHash},
		}, nil
	}

	// Fallback: no files in resolved state.
	fileName := path.Base(resolved.URL)
	return []FetchedFile{
		{RelPath: fileName, Content: content, SHA256: actualHash},
	}, nil
}

func (u *URLResolver) fetchURL(ctx context.Context, url, sourceName string) ([]byte, error) {
	if u.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, u.Timeout)
		defer cancel()
	}

	client := u.Client
	if client == nil {
		client = DefaultHTTPClient{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &SourceError{Source: sourceName, Operation: "fetch", Err: fmt.Errorf("creating request: %w", err)}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, &SourceError{Source: sourceName, Operation: "fetch", Err: fmt.Errorf("fetching %s: %w", url, err), Hint: "check network connectivity and URL"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &SourceError{
			Source:    sourceName,
			Operation: "fetch",
			Err:      fmt.Errorf("HTTP %d from %s", resp.StatusCode, url),
			Hint:     "check that the URL is accessible and returns the expected content",
		}
	}

	var reader io.Reader = resp.Body
	if u.MaxSize > 0 {
		reader = io.LimitReader(resp.Body, u.MaxSize+1)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, &SourceError{Source: sourceName, Operation: "fetch", Err: fmt.Errorf("reading response: %w", err)}
	}

	if u.MaxSize > 0 && int64(len(content)) > u.MaxSize {
		return nil, &SourceError{
			Source:    sourceName,
			Operation: "fetch",
			Err:      fmt.Errorf("file exceeds max size %d bytes", u.MaxSize),
			Hint:     "increase max_file_size or use a smaller file",
		}
	}

	return content, nil
}

func parseChecksum(checksum string) (algo, hash string, err error) {
	parts := strings.SplitN(checksum, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid checksum format '%s' — expected 'algorithm:hash' (e.g., 'sha256:abcdef...')", checksum)
	}
	return parts[0], parts[1], nil
}

func computeURLHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
