package annotation

import (
	"fmt"
	"go/ast"
)

type Gen struct {
}

var _ M = (*Gen)(nil)

func (Gen) Name() string {
	return "gen"
}

func (Gen) Match(node ast.Node) (err error) {
	if fd, ok := node.(*ast.FuncDecl); !ok || MethodReceiver(fd) != "" || fd.Name.Name != "Injects" {
		err = fmt.Errorf("the position of the `@Gen` annotation is incorrect, needed is `func Injects(container *sdk.Container) error`")
	}
	return
}

func (g Gen) As() (_ M) {
	return
}
