## 注解生成 & ioc 容器库

### 使用Ioc

```golang
// model.go
type A struct {

}

type B struct {
    *A
}

// @Inject(lazy="false")
func NewA() *A {
    return &A{}
}

// @Inject
func NewB(a *A) *B, error {
    return &B{a}, nil
}

// main.go
import (
    "github.com/iocgo/sdk"
)

// @Gen
func Injects(container sdk.Container) error {
   panic("auto implements")
}

func main() {
    container := sdk.NewContainer()
    err := Injects(container)
    if err != nil {
        panic(err)
    }

    err = container.Run()
    if err != nil {
        panic(err)
    }
}

// cmd/main.go
import (
    "github.com/iocgo/sdk/gen/tool"
)

func main() {
    tool.Exec()
}
```

### 使用代理
```go
// model.go
type A struct {

}

type IEcho interface {
    Echo()
}

// @Proxy(target="IEcho")
func IEchoInvocationHandler(ctx sdk.Context[IEcho]) {
    // before
    fmt.Println(ctx.In...)

    // instance & method name
    fmt.Println(ctx.Receiver, ctx.Name)

    // do method
    ctx.Do()

    // after
    fmt.Println(ctx.Out...)
}

func (A) Echo() {
    fmt.Println("A.Echo()")
}

// @Inject(name="model.A", lazy="false", proxy="model.IEcho")
func NewA() *A {
    return &A{}
}

// main.go
import (
    "github.com/iocgo/sdk"
)

// @Gen()
func Injects(container sdk.Container) error {
   panic("auto implements")
}

func main() {
    container := sdk.NewContainer()
    err := Injects(container)
    if err != nil {
        panic(err)
    }

    err = container.Run()
    if err != nil {
        panic(err)
    }

    bean, err := sdk.InvokeAs[model.Echo](container, "model.A")
    if err != nil {
        panic(err)
    }
    bean.Echo()
}
```



### 参考示例

1. [examples](examples/main.go)
2. [sdk-examples](https://github.com/iocgo/sdk-examples)
