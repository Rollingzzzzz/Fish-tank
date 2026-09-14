// G7.1: convention checker — verifies the repo follows README §2 (C2/C3):
//   - every internal package has a doc.go
//   - no non-test file exceeds ~300 lines
//   - the C2 import graph holds (no sideways/upward imports)
//   - F19: committed -shot evidence passes the uiaudit visibility gate
//
// Run: go run ./scripts/check
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// forbidden[srcPackageDir] lists package dirs that srcPackageDir may not import.
var forbidden = map[string][]string{
	"render":   {"sim", "ui", "agents", "content", "config", "game", "llm", "audio"},
	"sim":      {"render", "ui", "agents", "content", "config", "game", "llm", "audio"},
	"audio":    {"render", "sim", "ui", "agents", "content", "config", "game", "llm"},
	"ui":       {"sim", "agents", "llm", "audio", "config", "game"},
	"agents":   {"sim", "render", "ui", "audio", "config", "game"},
	"content":  {"sim", "render", "ui", "agents", "llm", "audio", "config", "game"},
	"config":   {"sim", "render", "ui", "agents", "llm", "audio", "content", "game"},
	"llm":      {"sim", "render", "ui", "agents", "audio", "content", "config", "game"},
	"contract": {"sim", "render", "ui", "agents", "audio", "content", "config", "llm", "game"},
}

func main() {
	fail := false

	// 1. doc.go presence + file length ceiling
	packageDirs := map[string]bool{}
	err := filepath.Walk("internal", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		dir := filepath.Dir(path)
		packageDirs[dir] = true
		if !strings.HasSuffix(path, "_test.go") && !strings.HasSuffix(path, "doc.go") {
			if n := countLines(path); n > 300 {
				fmt.Printf("VIOLATION: %s has %d lines (ceiling 300)\n", path, n)
				fail = true
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	for dir := range packageDirs {
		doc := filepath.Join(dir, "doc.go")
		if _, err := os.Stat(doc); err != nil {
			fmt.Printf("VIOLATION: %s has no doc.go\n", dir)
			fail = true
		}
	}

	// 2. import graph
	fset := token.NewFileSet()
	for dir := range packageDirs {
		pkgs, err := parser.ParseDir(fset, dir, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}
		base := filepath.Base(dir)
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for _, imp := range file.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					for _, bad := range forbidden[base] {
						if strings.HasSuffix(path, "/internal/"+bad) || path == "internal/"+bad {
							fmt.Printf("VIOLATION: %s imports %s (C2)\n", dir, path)
							fail = true
						}
					}
				}
			}
		}
	}

	// 3. F19 evidence gate: every committed -shot manifest must pass uiaudit
	// (all expected controls visible in the evidence PNGs). The gate only
	// activates once evidence directories exist, so WIP branches stay green.
	var manifestDirs []string
	_ = filepath.Walk("docs/art", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Name() != "manifest.json" {
			return nil
		}
		manifestDirs = append(manifestDirs, filepath.Dir(path))
		return nil
	})
	if len(manifestDirs) > 0 {
		sort.Strings(manifestDirs)
		cmd := exec.Command("go", append([]string{"run", "./scripts/uiaudit"}, manifestDirs...)...)
		out, err := cmd.CombinedOutput()
		fmt.Print(string(out))
		if err != nil {
			fmt.Println("VIOLATION: uiaudit evidence gate failed")
			fail = true
		}
	}

	if fail {
		fmt.Println("CHECK: FAIL")
		os.Exit(1)
	}
	fmt.Println("CHECK: all conventions hold (doc.go, line ceiling, import graph, evidence gate)")
}

// countLines counts \n occurrences (fast line estimate).
func countLines(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	return strings.Count(string(b), "\n")
}

// sortKeys is kept for deterministic future reporting.
func sortKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var _ = ast.Print
