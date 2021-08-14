package functionquickfix

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"unicode"
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
				if callExprName(s) == targetName {
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

		switch t := ty.(type) {
		case *types.Tuple:
			n := t.Len()
			for i := 0; i < n; i++ {
				stubArgs = append(stubArgs, Arg{
					Name: typeToArgName(t.At(i).Type()),
					Type: types.Default(t.At(i).Type()),
				})
			}
		default:
			// does the argument have a name we can reuse?
			// only happens in case of a *ast.Ident
			var name string
			if ident, ok := arg.(*ast.Ident); ok {
				name = ident.Name
			}

			if name == "" {
				name = typeToArgName(ty)
			}

			stubArgs = append(stubArgs, Arg{
				Name: name,
				Type: types.Default(ty),
			})
		}
	}

	stub := generateFuncStub(callExprName(callExpr), stubArgs)
	return stub, nil
}

func generateFuncStub(name string, args Args) string {
	return "func " + name + generateArgsListStub(args) + " {}"
}

func generateArgsListStub(args Args) string {
	uniqueArgs := ensureArgsUniqueness(args)
	s := "("

	for i, arg := range uniqueArgs {
		s += arg.Name + " " + arg.Type.String()
		if i != len(args)-1 {
			s += ", "
		}
	}

	s += ")"
	return s
}

func ensureArgsUniqueness(args Args) Args {
	// count all occurrences of the same arg name
	occurrences := map[string]int{}
	for _, arg := range args {
		occurrences[arg.Name]++
	}

	// prepare new names (aliases) for differrent
	// occurrences of the same arg name:
	// (a, a) -> (a1, a2)
	aliases := map[string][]string{}
	for name, occs := range occurrences {
		if occs <= 1 {
			continue
		}
		for i := 1; i <= occs; i++ {
			aliases[name] = append(aliases[name], name+fmt.Sprintf("%d", i))
		}
	}

	// build new list of args with unique names,
	// after using of the new names remove it from the pool.
	// If there are no aliases use the original identifier.
	var newArgs Args
	for _, arg := range args {
		aliasesForName, shouldUseAlias := aliases[arg.Name]
		if !shouldUseAlias {
			newArgs = append(newArgs, arg)
			continue
		}

		alias := aliasesForName[0]
		aliases[arg.Name] = aliases[arg.Name][1:]

		newArgs = append(newArgs, Arg{
			Name: alias,
			Type: arg.Type,
		})
	}

	return newArgs
}

type inspector func(ast.Node) bool

func (f inspector) Visit(node ast.Node) ast.Visitor {
	if f(node) {
		return f
	}
	return nil
}

func typeToArgName(ty types.Type) string {
	s := types.Default(ty).String()

	switch t := ty.(type) {
	case *types.Basic:
		// use first letter in type name for basic types
		return s[0:1]
	case *types.Slice:
		// use element type to decide var name for slices
		return typeToArgName(t.Elem())
	case *types.Array:
		// use element type to decide var name for arrays
		return typeToArgName(t.Elem())
	}

	s = strings.TrimLeft(s, "*") // if type is a pointer get rid of '*'

	if s == "error" {
		return "err"
	}

	// remove package (if present)
	// and make first letter lowercase
	parts := strings.Split(s, ".")
	a := []rune(parts[len(parts)-1])
	a[0] = unicode.ToLower(a[0])
	return string(a)
}

func callExprName(callExpr *ast.CallExpr) string {
	ident, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return ""
	}

	return ident.Name
}
