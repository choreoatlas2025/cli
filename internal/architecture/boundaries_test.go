// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These dependency checks protect execution responsibilities, not a directory
// naming convention. A lower carrier must not acquire an upper consumer.
func TestExecutionDependencies(t *testing.T) {
	allowed := map[string]string{
		"verdict":     "",
		"input":       "",
		"trace":       "input",
		"evidence":    "trace verdict",
		"spec":        "input trace",
		"validate":    "evidence input spec trace verdict",
		"discovery":   "spec trace validate",
		"result":      "input spec trace verdict",
		"baseline":    "fileio input result spec verdict",
		"report":      "fileio report/html result verdict",
		"report/html": "fileio result verdict",
	}
	const prefix = "github.com/choreoatlas2025/cli/internal/"
	for pkg, dependencies := range allowed {
		for _, file := range sourceFiles(t, pkg) {
			node := parse(t, file)
			for _, imp := range node.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(path, prefix) {
					continue
				}
				dependency := strings.TrimPrefix(path, prefix)
				if !strings.Contains(" "+dependencies+" ", " "+dependency+" ") {
					t.Errorf("%s imports %s outside its execution boundary", file, dependency)
				}
			}
		}
	}
}

func TestDraftGenerationDoesNotOwnPersistence(t *testing.T) {
	for _, file := range sourceFiles(t, "discovery") {
		node := parse(t, file)
		for _, imp := range node.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if path == "os" || path == "io" || path == "io/fs" || strings.HasSuffix(path, "/fileio") || strings.HasSuffix(path, "/cli") {
				t.Errorf("draft generation acquired side effects: %s imports %s", file, path)
			}
		}
		ast.Inspect(node, func(n ast.Node) bool {
			if call, ok := n.(*ast.SelectorExpr); ok {
				if pkg, ok := call.X.(*ast.Ident); ok && pkg.Name == "fmt" && strings.HasPrefix(call.Sel.Name, "Print") {
					t.Errorf("draft generation writes console output in %s", file)
				}
			}
			return true
		})
	}
}

func TestMatchersCannotEvaluateContracts(t *testing.T) {
	for _, name := range []string{"flow_match.go", "graph_match.go"} {
		file := filepath.Join("..", "validate", name)
		node := parse(t, file)
		ast.Inspect(node, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				switch id.Name {
				case "evaluation", "evaluateStep", "evaluateFlowMatches", "evaluateGraphMatches", "CompilePlan":
					t.Errorf("matcher entered evaluation boundary: %s uses %s", file, id.Name)
				}
			}
			return true
		})
		for _, imp := range node.Imports {
			if strings.Contains(imp.Path.Value, "cel-go") || strings.Contains(imp.Path.Value, "/evidence") {
				t.Errorf("matcher acquired projection or CEL in %s", file)
			}
		}
	}
}

func sourceFiles(t *testing.T, pkg string) []string {
	t.Helper()
	dir := filepath.Join("..", filepath.FromSlash(pkg))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	if len(files) == 0 {
		t.Fatalf("no implementation in %s", dir)
	}
	return files
}

func parse(t *testing.T, file string) *ast.File {
	t.Helper()
	node, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return node
}
