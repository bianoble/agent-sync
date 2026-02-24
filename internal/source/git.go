package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bianoble/agent-sync/internal/config"
)

// GitResolver resolves and fetches files from git repositories.
type GitResolver struct{}

func (g *GitResolver) Resolve(ctx context.Context, src config.Source, projectRoot string) (*ResolvedSource, error) {
	if src.Repo == "" {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("repo is required"), Hint: "add 'repo: https://...' to the source"}
	}
	if src.Ref == "" {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("ref is required"), Hint: "add 'ref: <tag-or-branch>'"}
	}

	// Clone to a temp directory.
	tmpDir, err := os.MkdirTemp("", "agent-sync-git-*")
	if err != nil {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("creating temp dir: %w", err)}
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Shallow clone with the specified ref.
	if cloneErr := gitClone(ctx, src.Repo, src.Ref, tmpDir); cloneErr != nil {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: cloneErr, Hint: "check repo URL, ref, and authentication"}
	}

	// Resolve commit SHA.
	commit, err := gitRevParse(ctx, tmpDir, "HEAD")
	if err != nil {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("resolving commit: %w", err)}
	}

	// Resolve tree SHA.
	tree, err := gitRevParse(ctx, tmpDir, "HEAD^{tree}")
	if err != nil {
		return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("resolving tree: %w", err)}
	}

	// Walk files and compute hashes.
	files := make(map[string]string)
	for _, pathFilter := range effectivePaths(src.Paths) {
		walkRoot := filepath.Join(tmpDir, pathFilter)
		info, statErr := os.Stat(walkRoot)
		if statErr != nil {
			continue // path doesn't exist in repo
		}

		if !info.IsDir() {
			// Single file.
			hash, hashErr := hashFile(walkRoot)
			if hashErr != nil {
				return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: hashErr}
			}
			files[pathFilter] = hash
			continue
		}

		err = filepath.Walk(walkRoot, func(path string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if fi.IsDir() {
				if strings.HasPrefix(fi.Name(), ".") && path != walkRoot {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(fi.Name(), ".") {
				return nil
			}
			rel, relErr := filepath.Rel(tmpDir, path)
			if relErr != nil {
				return relErr
			}
			hash, hashErr := hashFile(path)
			if hashErr != nil {
				return hashErr
			}
			files[rel] = hash
			return nil
		})
		if err != nil {
			return nil, &SourceError{Source: src.Name, Operation: "resolve", Err: fmt.Errorf("walking files: %w", err)}
		}
	}

	return &ResolvedSource{
		Name:   src.Name,
		Type:   "git",
		Commit: commit,
		Tree:   tree,
		Repo:   src.Repo,
		Files:  files,
	}, nil
}

func (g *GitResolver) Fetch(ctx context.Context, resolved *ResolvedSource) ([]FetchedFile, error) {
	if resolved.Repo == "" {
		return nil, &SourceError{Source: resolved.Name, Operation: "fetch", Err: fmt.Errorf("resolved source missing repo URL")}
	}

	tmpDir, err := os.MkdirTemp("", "agent-sync-git-fetch-*")
	if err != nil {
		return nil, &SourceError{Source: resolved.Name, Operation: "fetch", Err: err}
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Clone at the resolved commit.
	if cloneErr := gitCloneAtCommit(ctx, resolved.Repo, resolved.Commit, tmpDir); cloneErr != nil {
		return nil, &SourceError{Source: resolved.Name, Operation: "fetch", Err: cloneErr, Hint: "check repo access and commit SHA"}
	}

	var fetched []FetchedFile
	for relPath, expectedHash := range resolved.Files {
		absPath := filepath.Join(tmpDir, relPath)
		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return nil, &SourceError{Source: resolved.Name, Operation: "fetch", Err: fmt.Errorf("reading %s: %w", relPath, readErr)}
		}

		actualHash := computeSHA256(content)
		if actualHash != expectedHash {
			return nil, &SourceError{
				Source:    resolved.Name,
				Operation: "fetch",
				Err:       fmt.Errorf("hash mismatch for %s: expected %s, got %s", relPath, expectedHash, actualHash),
			}
		}

		fetched = append(fetched, FetchedFile{
			RelPath: relPath,
			Content: content,
			SHA256:  actualHash,
		})
	}

	return fetched, nil
}

func effectivePaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{"."}
	}
	return paths
}

func gitClone(ctx context.Context, repo, ref, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", ref, "--single-branch", repo, dest)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If --branch fails (e.g., for a commit SHA), try full clone + checkout.
		cmd2 := exec.CommandContext(ctx, "git", "clone", "--no-checkout", repo, dest)
		cmd2.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if _, err2 := cmd2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(output)), err)
		}
		cmd3 := exec.CommandContext(ctx, "git", "-C", dest, "checkout", ref)
		if out3, err3 := cmd3.CombinedOutput(); err3 != nil {
			return fmt.Errorf("git checkout %s failed: %s: %w", ref, strings.TrimSpace(string(out3)), err3)
		}
	}
	return nil
}

func gitCloneAtCommit(ctx context.Context, repo, commit, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--no-checkout", repo, dest)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	cmd2 := exec.CommandContext(ctx, "git", "-C", dest, "checkout", commit)
	if output, err := cmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s failed: %s: %w", commit, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func gitRevParse(ctx context.Context, repoDir, rev string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "rev-parse", rev)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return computeSHA256(data), nil
}

func computeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
