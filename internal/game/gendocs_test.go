package game

import (
	"os"
	"testing"
)

// TestDocsAreCurrent is the golden test that keeps the generated catalog honest:
// it regenerates the docs in memory and fails if the committed files differ, so
// content changes can't silently desync from the docs. Fix: `make docs` + commit.
func TestDocsAreCurrent(t *testing.T) {
	chdirToRepoRoot(t)
	for path, want := range GenerateDocs() {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s is missing — run `make docs` and commit it (%v)", path, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s is out of date — run `make docs` and commit the result", path)
		}
	}
}
