package functionquickfix_test

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

const undeclaredName = "u"

const primitiveType = `
package missingfunction

func a(s string) {
	u(s)
}
`

const argumentsWithSameName = `
package missingfunction

func a(s string, i int) {
	u(s, i, s)
}
`

const errorAsArgument = `
package missingfunction

func a() {
	var err error
	u(err)
}
`

const functionReturningMultipleValuesAsArgument = `
package missingfunction

func a() {
	u(b())
}

func b() (string, error) {
	return "", nil
}
`

const argumentsWithoutIdentifiers = `
package missingfunction

type T struct {}

func a() {
	u("hey compiler", T{}, &T{})
}
`

const returnedValuesAsArguments = `
package missingfunction

import "io"

func a() {
	u(io.MultiReader())
}
`

const importedType = `
package missingfunction

import "io"

func a(r io.Reader) {
	u(r)
}
`

const nonPrimitiveType = `
package missingfunction

type T struct {}

func a() {
	pointer := &T{}
	var value T
	u(pointer, value)
}
`

const operationsAsArguments = `
package missingfunction

import "time"

func a() {
	u(10 * time.Second)
}
`

const sliceAsArgument = `
package missingfunction

func a() {
	u([]int{1, 2})
}
`

func TestFunctionQuickfix(t *testing.T) {
	sources := []string{
		primitiveType,
		importedType,
		nonPrimitiveType,
		argumentsWithSameName,
		argumentsWithoutIdentifiers,
		returnedValuesAsArguments,
		errorAsArgument,
		functionReturningMultipleValuesAsArgument,
		operationsAsArguments,
		sliceAsArgument,
	}

	for _, src := range sources {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "src.go", src, parser.AllErrors)
		if err != nil {
			t.Fatal(err)
		}

		shouldHaveUndeclaredName(t, fset, f, undeclaredName)

		stub, err := functionquickfix.GenerateFunctionStub(undeclaredName, src)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(stub)

		newSource := src + "\n" + stub
		fset = token.NewFileSet()
		f, err = parser.ParseFile(fset, "src.go", newSource, parser.AllErrors)
		if err != nil {
			t.Fatal(err, newSource)
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
