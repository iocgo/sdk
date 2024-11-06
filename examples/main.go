package main

import (
	"fmt"
	"github.com/iocgo/sdk"
	"github.com/iocgo/sdk/examples/inter"
	"github.com/iocgo/sdk/proxy"

	_ "github.com/gin-gonic/gin"
	_ "github.com/iocgo/sdk/router"
)

type Interface2 interface {
	inter.Interface1
	Hi()
}

// @Proxy(target="main.Interface2")
func Interface2InvocationHandler(ctx *proxy.Context[Interface2]) {
}

type StructA struct {
	Num int
}

func (StructA) Echo() {
	fmt.Println("A.Echo()")
}

func (StructA) Hi() {
	fmt.Println("A.Hi()")
}

var _ Interface2 = (*StructA)(nil)

// @Inject(config="{a: \"hi\"}")
func NewStructA(data sdk.AnnotationBytes) *StructA {
	return &StructA{}
}

func main() {

}
