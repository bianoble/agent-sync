package lock

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// specExampleLockfile is the example from spec Section 4.2.
const specExampleLockfile = `
version: 1

sources:

  - name: base-rules
    type: git
    repo: https://github.com/org/rules.git

    resolved:

      commit: 3f8c9abf
      tree: a8bcdef

      files:

        core/security.md:
          sha256: 123abc

        core/general.md:
          sha256: 456def

    status: ok

  - name: security-policy
    type: url

    resolved:

      url: https://example.com/policy.md
      sha256: abcdef123

    status: ok

  - name: team-standards
    type: local

    resolved:

      path: ./agents/standards/

      files:

        standards/naming.md:
          sha256: 789ghi

        standards/testing.md:
          sha256: 012jkl

    status: ok
`

func TestLockfileParseSpecExample(t *testing.T) {
	var lf Lockfile
	if err := yaml.Unmarshal([]byte(specExampleLockfile), &lf); err != nil {
		t.Fatalf("failed to parse spec example lockfile: %v", err)
	}

	if lf.Version != 1 {
		t.Errorf("version = %d, want 1", lf.Version)
	}

	if len(lf.Sources) != 3 {
		t.Fatalf("sources count = %d, want 3", len(lf.Sources))
	}

	// Git source.
	git := lf.Sources[0]
	if git.Name != "base-rules" {
		t.Errorf("sources[0].name = %q", git.Name)
	}
	if git.Type != "git" {
		t.Errorf("sources[0].type = %q", git.Type)
	}
	if git.Repo != "https://github.com/org/rules.git" {
		t.Errorf("sources[0].repo = %q", git.Repo)
	}
	if git.Resolved.Commit != "3f8c9abf" {
		t.Errorf("sources[0].resolved.commit = %q", git.Resolved.Commit)
	}
	if git.Resolved.Tree != "a8bcdef" {
		t.Errorf("sources[0].resolved.tree = %q", git.Resolved.Tree)
	}
	if len(git.Resolved.Files) != 2 {
		t.Fatalf("sources[0].resolved.files count = %d, want 2", len(git.Resolved.Files))
	}
	if git.Resolved.Files["core/security.md"].SHA256 != "123abc" {
		t.Errorf("sources[0] core/security.md sha256 = %q", git.Resolved.Files["core/security.md"].SHA256)
	}
	if git.Status != "ok" {
		t.Errorf("sources[0].status = %q", git.Status)
	}

	// URL source.
	url := lf.Sources[1]
	if url.Name != "security-policy" {
		t.Errorf("sources[1].name = %q", url.Name)
	}
	if url.Resolved.URL != "https://example.com/policy.md" {
		t.Errorf("sources[1].resolved.url = %q", url.Resolved.URL)
	}
	if url.Resolved.SHA256 != "abcdef123" {
		t.Errorf("sources[1].resolved.sha256 = %q", url.Resolved.SHA256)
	}

	// Local source.
	local := lf.Sources[2]
	if local.Name != "team-standards" {
		t.Errorf("sources[2].name = %q", local.Name)
	}
	if local.Resolved.Path != "./agents/standards/" {
		t.Errorf("sources[2].resolved.path = %q", local.Resolved.Path)
	}
	if len(local.Resolved.Files) != 2 {
		t.Fatalf("sources[2].resolved.files count = %d, want 2", len(local.Resolved.Files))
	}
}

func TestLockfileRoundTrip(t *testing.T) {
	original := Lockfile{
		Version: 1,
		Sources: []LockedSource{
			{
				Name: "test",
				Type: "url",
				Resolved: ResolvedState{
					URL:    "https://example.com/file.md",
					SHA256: "abc123",
				},
				Status: "ok",
			},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundTripped Lockfile
	if err := yaml.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if roundTripped.Version != 1 {
		t.Errorf("version = %d, want 1", roundTripped.Version)
	}
	if len(roundTripped.Sources) != 1 {
		t.Fatalf("sources count = %d, want 1", len(roundTripped.Sources))
	}
	if roundTripped.Sources[0].Resolved.SHA256 != "abc123" {
		t.Errorf("resolved.sha256 = %q", roundTripped.Sources[0].Resolved.SHA256)
	}
}

func TestLockfileUnknownFieldsIgnored(t *testing.T) {
	input := `
version: 1
future_field: should be ignored
sources:
  - name: test
    type: local
    resolved:
      path: ./test/
    status: ok
    new_field: also ignored
`
	var lf Lockfile
	if err := yaml.Unmarshal([]byte(input), &lf); err != nil {
		t.Fatalf("should ignore unknown fields: %v", err)
	}
	if lf.Version != 1 {
		t.Errorf("version = %d, want 1", lf.Version)
	}
}
