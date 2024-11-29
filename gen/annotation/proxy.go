package annotation

import (
	"fmt"
	"go/ast"
)

type Proxy struct {
	Target string `annotation:"name=target,default="`
	Scan   string `annotation:"name=scan,default="`
	Igm    string `annotation:"name=igm,default="`
}

var _ M = (*Proxy)(nil)

func (p Proxy) Name() string {
	return "proxy"
}

func (p Proxy) Match(node ast.Node) (err error) {
	if p.Target == "" {
		return fmt.Errorf("please specify the proxy function")
	}

	if fd, ok := node.(*ast.FuncDecl); !ok || MethodReceiver(fd) != "" {
		err = fmt.Errorf("the position of the `@Proxy` annotation is incorrect, needed `func( ctx *proxy.Context )`")
	}
	return
}

func (p Proxy) As() (_ M) {
	return
}
