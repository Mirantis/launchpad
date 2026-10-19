package configurer_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guardedBuilders are the command builders whose output is handed to a remote
// shell. Any non-constant argument interpolated into one of them is executable
// on the target host, so it has to go through a quoting builder (DockerCmd,
// sh.Command, ps.SingleQuote) instead of a raw format string.
var guardedBuilders = map[string]struct{}{
	"DockerCommandf": {},
}

// TestRemoteCommandsAreNotBuiltByRawSprintf fails when a remote command is
// assembled from a format string with a non-constant argument. Constant
// arguments are allowed: they cannot carry attacker-controlled data.
func TestRemoteCommandsAreNotBuiltByRawSprintf(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	fset := token.NewFileSet()

	var violations []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if base := entry.Name(); base == ".git" || base == "vendor" || base == ".worktrees" {
				return filepath.SkipDir
			}

			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}

		consts := constIdents(file)

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if _, guarded := guardedBuilders[sel.Sel.Name]; !guarded {
				return true
			}
			// Only the interpolated arguments matter; the template itself is
			// written in the source.
			for _, arg := range call.Args[1:] {
				if isConstantExpr(arg, consts) {
					continue
				}
				rel, _ := filepath.Rel(root, path)
				violations = append(violations, fmt.Sprintf("%s:%d %s", rel, fset.Position(arg.Pos()).Line, sel.Sel.Name))
			}

			return true
		})

		return nil
	})
	if err != nil {
		t.Fatalf("walking repository: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("remote commands built from a format string with non-constant arguments; use Configurer.DockerCmd() instead:\n  %s", strings.Join(violations, "\n  "))
	}
}

// constIdents collects the names declared in const blocks of a file so that
// `DockerCommandf(someConst)` is not reported.
func constIdents(file *ast.File) map[string]struct{} {
	names := map[string]struct{}{}

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				names[name.Name] = struct{}{}
			}
		}
	}

	return names
}

// isConstantExpr reports whether an expression is a literal, a local constant
// or a concatenation of those.
func isConstantExpr(expr ast.Expr, consts map[string]struct{}) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		_, ok := consts[e.Name]

		return ok
	case *ast.SelectorExpr:
		// Package-level constants from another package, e.g. constant.ManagedLabelCmd.
		// Those are source-controlled; host data never arrives this way.
		return isExportedConstRef(e)
	case *ast.BinaryExpr:
		return isConstantExpr(e.X, consts) && isConstantExpr(e.Y, consts)
	case *ast.ParenExpr:
		return isConstantExpr(e.X, consts)
	default:
		return false
	}
}

// isExportedConstRef reports whether a selector looks like pkg.Name, which the
// guard treats as a source constant rather than runtime data. Method calls and
// field accesses on values are not selectors of this shape in practice because
// those appear as CallExpr or have a non-ident receiver chain.
func isExportedConstRef(sel *ast.SelectorExpr) bool {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Obj == nil && strings.ToUpper(sel.Sel.Name[:1]) == sel.Sel.Name[:1]
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test directory")
		}
		dir = parent
	}
}
