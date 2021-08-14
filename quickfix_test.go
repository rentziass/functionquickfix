package functionquickfix_test

// TODO: uniqueness in generated arguments names
// TODO: use non-primitive types

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

type Arg struct {
	Name string
	Type types.Type
}

type Args []Arg

const missingFunctionPrimitiveType = `
package missingfunction

func a(s string) {
	b(s)
}
`

const missingFunctionNonPrimitiveType = `
package missingfunction

import "io"

func a(r io.Reader) {
	z(r)
}
`

const missingFunctionCustomType = `
package missingfunction

type T struct {}

func a() {
	pointer := &T{}
	var value T
	f(pointer, value)
}
`

func TestFunctionQuickfix(t *testing.T) {
	tests := []struct {
		source         string
		undeclaredName string
	}{
		{
			source:         missingFunctionPrimitiveType,
			undeclaredName: "b",
		},
		{
			source:         missingFunctionNonPrimitiveType,
			undeclaredName: "z",
		},
		{
			source:         missingFunctionCustomType,
			undeclaredName: "f",
		},
	}

	for _, testCase := range tests {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "src.go", testCase.source, parser.AllErrors)
		if err != nil {
			t.Fatal(err)
		}

		shouldHaveUndeclaredName(t, fset, f, testCase.undeclaredName)

		var callExpr *ast.CallExpr
		var args []ast.Expr

		newInspector := func(targetName string) inspector {
			return func(n ast.Node) bool {
				switch s := n.(type) {
				case *ast.CallExpr:
					if exprToString(s.Fun) == targetName {
						callExpr = s
						args = s.Args
					}

					return false
				}
				return true
			}
		}

		ast.Walk(newInspector(testCase.undeclaredName), f)

		// found all types, don't stop at error
		info := types.Info{
			Types: map[ast.Expr]types.TypeAndValue{},
		}
		conf := types.Config{
			Importer: importer.Default(),
			Error: func(err error) {
			},
		}
		_, _ = conf.Check("fib", fset, []*ast.File{f}, &info)

		var stubArgs Args
		for _, arg := range args {
			ty := info.TypeOf(arg)
			if ty == nil {
				t.Fatalf("nil type for arg %v", arg)
			}

			stubArgs = append(stubArgs, Arg{
				Name: exprToString(arg),
				Type: ty,
			})
		}

		stub := generateFuncStub(exprToString(callExpr.Fun), stubArgs)
		fmt.Println(stub)

		newSource := testCase.source + "\n" + stub
		fset = token.NewFileSet()
		f, err = parser.ParseFile(fset, "src.go", newSource, parser.AllErrors)
		if err != nil {
			t.Fatal(err)
		}
		shouldNotHaveErrors(t, fset, f)
	}
}

type inspector func(ast.Node) bool

func (f inspector) Visit(node ast.Node) ast.Visitor {
	if f(node) {
		return f
	}
	return nil
}

func shouldHaveUndeclaredName(t *testing.T, fset *token.FileSet, f *ast.File, name string) {
	info := types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	conf := types.Config{
		Importer: importer.Default(),
	}
	_, err := conf.Check("fib", fset, []*ast.File{f}, &info)
	if err == nil {
		t.Fatal("there should have been an error")
	}
	if !strings.Contains(err.Error(), "undeclared name: "+name) {
		t.Fatalf("%s function should be undeclared, got %s", name, err.Error())
	}
}

func generateFuncStub(name string, args Args) string {
	return "func " + name + generateArgsListStub(args) + " {}"
}

func generateArgsListStub(args Args) string {
	s := "("

	for i, arg := range args {
		s += arg.Name + " " + arg.Type.String()
		if i != len(args)-1 {
			s += ", "
		}
	}

	s += ")"
	return s
}

func exprToString(expr ast.Expr) string {
	return fmt.Sprintf("%v", expr)
}

func shouldNotHaveErrors(t *testing.T, fset *token.FileSet, f *ast.File) {
	info := types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	conf := types.Config{
		Importer: importer.Default(),
	}
	_, err := conf.Check("fib", fset, []*ast.File{f}, &info)
	if err != nil {
		t.Fatalf("expected no error, got %s", err.Error())
	}
}
