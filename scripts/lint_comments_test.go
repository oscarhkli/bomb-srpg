package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func buildChecker(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "lint-comments")
	if out, err := exec.Command("go", "build", "-o", bin, "lint-comments.go").CombinedOutput(); err != nil {
		t.Fatalf("build checker: %v\n%s", err, out)
	}
	return bin
}

func TestLintComments(t *testing.T) {
	bin := buildChecker(t)
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "three-line block passes",
			src:  "package x\n\n// a\n// b\n// c\nfunc f() {}\n",
			want: nil,
		},
		{
			name: "four-line block fails",
			src:  "package x\n\n// a\n// b\n// c\n// d\nfunc f() {}\n",
			want: []string{"x.go:3"},
		},
		{
			name: "block comment over three lines fails",
			src:  "package x\n\n/*\na\nb\nc\n*/\nfunc f() {}\n",
			want: []string{"x.go:3"},
		},
		{
			name: "trailing comment fails",
			src:  "package x\n\nvar v = 1 // why\n",
			want: []string{"x.go:3"},
		},
		{
			name: "go directive passes",
			src:  "//go:build ignore\n\npackage x\n",
			want: nil,
		},
		{
			name: "directive does not count towards the block",
			src:  "package x\n\n//go:generate echo\n// a\n// b\n// c\nfunc f() {}\n",
			want: nil,
		},
		{
			name: "nolint with a reason fails",
			src:  "package x\n\n//nolint:gosec // why\nfunc f() {}\n",
			want: []string{"x.go:3"},
		},
		{
			name: "slashes inside a string literal pass",
			src:  "package x\n\nvar v = \"http://example.com // not a comment\"\n",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte(tt.src), 0o600); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(bin, dir).CombinedOutput()
			var got []string
			for line := range strings.Lines(string(out)) {
				loc, _, _ := strings.Cut(strings.TrimPrefix(line, dir+string(filepath.Separator)), ": ")
				got = append(got, loc)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("locations = %v, want %v\noutput:\n%s", got, tt.want, out)
			}
			if failed := err != nil; failed != (len(tt.want) > 0) {
				t.Errorf("exit error = %v, want failure %v", err, len(tt.want) > 0)
			}
		})
	}
}
