package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bianoble/agent-sync/internal/config"
)

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestURLResolverSuccess(t *testing.T) {
	content := []byte("# Security Policy\n")
	hash := sha256Hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	r := &URLResolver{}
	src := config.Source{
		Name:     "policy",
		Type:     "url",
		URL:      srv.URL + "/policy.md",
		Checksum: "sha256:" + hash,
	}

	resolved, err := r.Resolve(context.Background(), src, t.TempDir())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if resolved.Name != "policy" {
		t.Errorf("name = %q", resolved.Name)
	}
	if resolved.URL != src.URL {
		t.Errorf("url = %q", resolved.URL)
	}
	if len(resolved.Files) != 1 {
		t.Errorf("files count = %d", len(resolved.Files))
	}
	if resolved.Files["policy.md"] != hash {
		t.Errorf("hash = %q, want %q", resolved.Files["policy.md"], hash)
	}
}

func TestURLResolverChecksumMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("actual content"))
	}))
	defer srv.Close()

	r := &URLResolver{}
	src := config.Source{
		Name:     "bad",
		Type:     "url",
		URL:      srv.URL + "/file.md",
		Checksum: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}

	_, err := r.Resolve(context.Background(), src, t.TempDir())
	if err == nil {
		t.Fatal("expected error for checksum mismatch")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestURLResolverHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := &URLResolver{}
	src := config.Source{
		Name:     "missing",
		Type:     "url",
		URL:      srv.URL + "/gone.md",
		Checksum: "sha256:abc",
	}

	_, err := r.Resolve(context.Background(), src, t.TempDir())
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestURLResolverMaxSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 1024))
	}))
	defer srv.Close()

	r := &URLResolver{MaxSize: 100}
	src := config.Source{
		Name:     "big",
		Type:     "url",
		URL:      srv.URL + "/large.bin",
		Checksum: "sha256:abc",
	}

	_, err := r.Resolve(context.Background(), src, t.TempDir())
	if err == nil {
		t.Fatal("expected error for file too large")
	}
	if !strings.Contains(err.Error(), "max size") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestURLResolverMissingURL(t *testing.T) {
	r := &URLResolver{}
	_, err := r.Resolve(context.Background(), config.Source{Name: "test", Checksum: "sha256:abc"}, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestURLResolverMissingChecksum(t *testing.T) {
	r := &URLResolver{}
	_, err := r.Resolve(context.Background(), config.Source{Name: "test", URL: "https://example.com/file.md"}, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing checksum")
	}
}

func TestURLResolverUnsupportedAlgorithm(t *testing.T) {
	r := &URLResolver{}
	src := config.Source{
		Name:     "test",
		URL:      "https://example.com/file.md",
		Checksum: "md5:abc123",
	}
	_, err := r.Resolve(context.Background(), src, t.TempDir())
	if err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
	if !strings.Contains(err.Error(), "unsupported checksum algorithm") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestURLResolverInvalidChecksumFormat(t *testing.T) {
	r := &URLResolver{}
	src := config.Source{
		Name:     "test",
		URL:      "https://example.com/file.md",
		Checksum: "nocolon",
	}
	_, err := r.Resolve(context.Background(), src, t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid checksum format")
	}
	if !strings.Contains(err.Error(), "invalid checksum format") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestURLFetch(t *testing.T) {
	content := []byte("fetched content")
	hash := sha256Hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	r := &URLResolver{}
	resolved := &ResolvedSource{
		Name:  "test",
		Type:  "url",
		URL:   srv.URL + "/file.md",
		Files: map[string]string{"file.md": hash},
	}

	fetched, err := r.Fetch(context.Background(), resolved)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(fetched) != 1 {
		t.Fatalf("expected 1 file, got %d", len(fetched))
	}
	if string(fetched[0].Content) != "fetched content" {
		t.Errorf("content = %q", string(fetched[0].Content))
	}
}

func TestURLResolverTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("slow"))
	}))
	defer srv.Close()

	r := &URLResolver{Timeout: 100 * time.Millisecond}
	src := config.Source{
		Name:     "slow",
		Type:     "url",
		URL:      srv.URL + "/slow.md",
		Checksum: "sha256:abc",
	}

	_, err := r.Resolve(context.Background(), src, t.TempDir())
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
