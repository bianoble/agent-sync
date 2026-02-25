package cmd

import "testing"

func TestHumanSize(t *testing.T) {
	tests := []struct {
		want  string
		bytes int64
	}{
		{"0 B", 0},
		{"1 B", 1},
		{"512 B", 512},
		{"1023 B", 1023},
		{"1.0 KB", 1024},
		{"1.5 KB", 1536},
		{"1.0 MB", 1048576},
		{"1.0 GB", 1073741824},
		{"2.5 GB", 2684354560},
	}

	for _, tt := range tests {
		got := humanSize(tt.bytes)
		if got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
