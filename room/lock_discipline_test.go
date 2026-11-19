package room

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
)

type methodFacts struct {
	touchesMatch bool
	locks        bool
	callers      []string
}

func roomMethodFacts(t *testing.T) map[string]*methodFacts {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	fset := token.NewFileSet()
	facts := map[string]*methodFacts{}
	var decls []*ast.FuncDecl
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s) error = %v", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil || !isRoomReceiver(fn.Recv.List[0].Type) {
				continue
			}
			facts[fn.Name.Name] = &methodFacts{}
			decls = append(decls, fn)
		}
	}
	for _, fn := range decls {
		recv := fn.Recv.List[0].Names[0].Name
		self := facts[fn.Name.Name]
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if isIdent(sel.X, recv) && (sel.Sel.Name == "match" || sel.Sel.Name == "lastActivity") {
				self.touchesMatch = true
			}
			if inner, ok := sel.X.(*ast.SelectorExpr); ok && isIdent(inner.X, recv) && inner.Sel.Name == "mu" &&
				(sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock") {
				self.locks = true
			}
			if isIdent(sel.X, recv) {
				if callee, ok := facts[sel.Sel.Name]; ok {
					callee.callers = append(callee.callers, fn.Name.Name)
					if sel.Sel.Name == "read" || sel.Sel.Name == "command" {
						self.locks = true
					}
				}
			}
			return true
		})
	}
	return facts
}

func isRoomReceiver(expr ast.Expr) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	return isIdent(expr, "Room")
}

func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

func lockViolations(facts map[string]*methodFacts) []string {
	var bad []string
	for name, f := range facts {
		if !f.touchesMatch || f.locks {
			continue
		}
		if !strings.HasSuffix(name, "Locked") {
			bad = append(bad, name+" touches the match without the room lock")
			continue
		}
		for _, caller := range f.callers {
			if c := facts[caller]; !c.locks && !strings.HasSuffix(caller, "Locked") {
				bad = append(bad, name+" is called by "+caller+", which does not hold the lock")
			}
		}
		if len(f.callers) == 0 {
			bad = append(bad, name+" has no caller that holds the lock")
		}
	}
	slices.Sort(bad)
	return bad
}

func TestRoom_LockDiscipline(t *testing.T) {
	facts := roomMethodFacts(t)
	if len(facts) == 0 {
		t.Fatal("found no methods of Room")
	}
	if bad := lockViolations(facts); len(bad) > 0 {
		t.Errorf("lock discipline broken: %v", bad)
	}
}
