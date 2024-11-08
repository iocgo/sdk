package annotation

import (
	"fmt"
	"go/ast"
	"unicode"
)

type Inject struct {
	IsLazy     bool   `annotation:"name=lazy,default=true"`
	N          string `annotation:"name=name,default="`
	Alias      string `annotation:"name=alias,default="`
	Singleton  bool   `annotation:"name=singleton,default=true"`
	Initialize string `annotation:"name=init,default="`
	Qualifier  string `annotation:"name=qualifier,default="`
	Config     string `annotation:"name=config,default="`
}

var _ M = (*Inject)(nil)

func (Inject) Name() string {
	return "inject"
}

func (i Inject) Match(node ast.Node) (err error) {
	if i.Initialize != "" && !unicode.IsUpper([]rune(i.Initialize)[0]) {
		err = fmt.Errorf("the `@Inject(init)` value needs to start with a capital case")
		return
	}

	if _, ok := node.(*ast.FuncDecl); !ok {
		err = fmt.Errorf("the position of the `@Inject` annotation is incorrect, needed is function (ast.FuncDecl)")
	}
	return
}

func (i Inject) As() (_ M) {
	return
}
