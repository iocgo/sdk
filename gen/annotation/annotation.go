package annotation

import (
	"go/ast"
)

type M interface {
	Name() string
	Match(node ast.Node) error
	As() M
}

type Anon struct {
}

func (g *Anon) Name() string {
	return "anon"
}

func (g *Anon) Match(node ast.Node) error {
	var m M = g
	if n := g.As(); n != nil {
		m = n
	}
	if m == g {
		return nil
	}
	return m.Match(node)
}

func (g *Anon) As() M { return nil }

func MethodReceiver(decl *ast.FuncDecl) string {
	if decl.Recv == nil {
		return ""
	}

	for _, v := range decl.Recv.List {
		switch rv := v.Type.(type) {
		case *ast.Ident:
			return rv.Name
		case *ast.StarExpr:
			return rv.X.(*ast.Ident).Name
		case *ast.UnaryExpr:
			return rv.X.(*ast.Ident).Name
		}
	}
	return ""
}
