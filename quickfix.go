package functionquickfix

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

type Arg struct {
	Name string
	Type types.Type
}

type Args []Arg

func GenerateFunctionStub(undeclaredName string, source string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", source, 0)
	if err != nil {
		return "", err
	}

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

	ast.Walk(newInspector(undeclaredName), f)

	// found all types, don't stop at error
	info := types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
	}
	conf := types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
		},
	}
	_, _ = conf.Check("", fset, []*ast.File{f}, &info)

	var stubArgs Args
	for _, arg := range args {
		ty := info.TypeOf(arg)
		if ty == nil {
			return "", fmt.Errorf("could not determine type of arg %v", arg)
		}

		stubArgs = append(stubArgs, Arg{
			Name: exprToString(arg),
			Type: ty,
		})
	}

	stub := generateFuncStub(exprToString(callExpr.Fun), stubArgs)
	return stub, nil
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

type inspector func(ast.Node) bool

func (f inspector) Visit(node ast.Node) ast.Visitor {
	if f(node) {
		return f
	}
	return nil
}
