// Command gendocs regenerates docs/ from the game's content registries. Run via
// `make docs`. The generated files are checked by a golden test, so they must be
// committed whenever content changes.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"dmud/internal/game"
)

func main() {
	docs := game.GenerateDocs()
	paths := make([]string, 0, len(docs))
	for p := range docs {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, []byte(docs[path]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
}
