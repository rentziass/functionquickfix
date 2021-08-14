package functionquickfix_test

// TODO: uniqueness in generated arguments names

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/rentziass/functionquickfix"
)

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

		stub, err := functionquickfix.GenerateFunctionStub(testCase.undeclaredName, testCase.source)
		if err != nil {
			t.Fatal(err)
		}
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
