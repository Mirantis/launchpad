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
			// Every argument is checked, the template included: a runtime
			// template such as DockerCommandf(preformattedCommand) carries the
			// same injection as a runtime parameter.
			for _, arg := range call.Args {
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

// isConstantExpr reports whether an expression is a string literal, a constant
// declared in the same file, or a concatenation of those.
//
// Anything the guard cannot resolve to a constant declaration is treated as
// runtime data. In particular a selector such as pkg.Name or value.Field is
// rejected: without type resolution a package constant and a package-level
// variable or a struct field read at runtime are indistinguishable, and a
// false negative here is an unquoted command argument on a remote host. The
// cost is that a genuine cross-package constant has to be bound to a local
// const or moved into a DockerCmd().Raw(...) call.
func isConstantExpr(expr ast.Expr, consts map[string]struct{}) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Kind == token.STRING
	case *ast.Ident:
		_, ok := consts[e.Name]

		return ok
	case *ast.BinaryExpr:
		return isConstantExpr(e.X, consts) && isConstantExpr(e.Y, consts)
	case *ast.ParenExpr:
		return isConstantExpr(e.X, consts)
	default:
		return false
	}
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

// TestIsConstantExprRejectsRuntimeExpressions pins the inputs the guard must
// treat as runtime data. A false positive here silently disarms the guard.
func TestIsConstantExprRejectsRuntimeExpressions(t *testing.T) {
	t.Parallel()

	const src = `package p

const localConst = "node ls"

var pkgVar = struct{ Field string }{}

func f(runtimeCommand string) {
	_ = localConst
	_ = pkgVar
	_ = runtimeCommand
}
`

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	consts := constIdents(file)

	for _, tc := range []struct {
		name      string
		expr      string
		wantConst bool
	}{
		{name: "string literal", expr: `"node ls"`, wantConst: true},
		{name: "file-local constant", expr: "localConst", wantConst: true},
		{name: "concatenated literals", expr: `"node " + "ls"`, wantConst: true},
		{name: "runtime parameter", expr: "runtimeCommand", wantConst: false},
		{name: "imported selector", expr: "os.Args", wantConst: false},
		{name: "field on a package variable", expr: "pkgVar.Field", wantConst: false},
		{name: "call result", expr: "fmt.Sprintf(\"rm %s\", nodeID)", wantConst: false},
		{name: "concatenation with runtime value", expr: `"rm " + nodeID`, wantConst: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expr, err := parser.ParseExpr(tc.expr)
			if err != nil {
				t.Fatalf("parsing %q: %v", tc.expr, err)
			}
			if got := isConstantExpr(expr, consts); got != tc.wantConst {
				t.Fatalf("isConstantExpr(%q) = %v, want %v", tc.expr, got, tc.wantConst)
			}
		})
	}
}
