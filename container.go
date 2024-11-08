package sdk

import (
	"errors"
	"fmt"
	"github.com/iocgo/sdk/proxy"
	"github.com/iocgo/sdk/runtime"
	"github.com/samber/do/v2"
	"os"
	"os/signal"
	run "runtime"
	"slices"
	"strings"
	"sync"
)

type keys struct {
	sync.Mutex
	g []string
}

type Initializer interface {
	Init(*Container) error
	Order() int
}

type singleInitializer struct {
	order int
	init  func(*Container) error
}

type Container struct {
	inject *do.RootScope
	alias  map[string]string
	init   []func() error
}

var (
	threadLocal = runtime.NewThreadLocal[*keys](func() *keys {
		return &keys{}
	})
)

func (k *keys) push(key string) (re bool) {
	k.Lock()
	defer k.Unlock()

	if !slices.Contains(k.g, key) {
		re = true
	}

	k.g = append(k.g, key)
	return
}

func (i singleInitializer) Init(container *Container) (err error) {
	if i.init == nil {
		return
	}
	return i.init(container)
}

func (i singleInitializer) Order() int {
	return i.order
}

func NewContainer() *Container {
	return &Container{
		inject: do.New(),
		alias:  make(map[string]string),
	}
}

func InitializedWrapper(order int, init func(*Container) error) Initializer {
	return &singleInitializer{order, init}
}

func (c *Container) AddInitialized(i func() error) {
	c.init = append(c.init, i)
}

func (c *Container) Run(signals ...os.Signal) (err error) {
	beans := ListInvokeAs[Initializer](c)
	beans = append(beans, &singleInitializer{999, func(container *Container) (iErr error) {
		for _, exec := range c.init {
			if iErr = exec(); iErr != nil {
				return iErr
			}
		}
		return
	}})

	if len(beans) > 0 {
		slices.SortFunc[[]Initializer](beans, func(a, b Initializer) int {
			return elseOf(a.Order() == b.Order(), 0, elseOf(a.Order() > b.Order(), 1, -1))
		})

		for _, bean := range beans {
			if err = bean.Init(c); err != nil {
				return
			}
		}
	}

	// Logger

	// TODO -
	if len(signals) > 0 {
		w := make(chan os.Signal, 1)
		signal.Notify(w, signals...)
		<-w
	}
	return
}

func (c *Container) Inject() *do.RootScope {
	return c.inject
}

func (c *Container) Alias(name, fullName string) {
	if n, ok := c.alias[name]; ok {
		panic("alias '" + n + "' already exists")
	}
	c.alias[name] = fullName
}

func (c *Container) HealthLogger() string {
	injector := do.ExplainInjector(c.inject)
	return injector.String()
}

func (c *Container) Stop() (err error) {
	err = c.inject.Shutdown()
	if err != nil {
		return
	}

	// TODO
	return
}

func NameOf[T any]() string {
	return do.NameOf[T]()
}

func ProvideBean[T any](container *Container, name string, provider func() (T, error)) {
	do.ProvideNamed[T](container.inject, name, func(i do.Injector) (T, error) {
		return provider()
	})
}

func ProvideTransient[T any](container *Container, name string, provider func() (T, error)) {
	do.ProvideNamedTransient[T](container.inject, name, func(i do.Injector) (T, error) {
		return provider()
	})
}

func OverrideBean[T any](container *Container, name string, provider func() (T, error)) {
	do.OverrideNamed[T](container.inject, name, func(i do.Injector) (T, error) {
		return provider()
	})
}

func InvokeBean[T any](container *Container, name string) (t T, err error) {
	if name != "" {
		for {
			if n, ok := container.alias[name]; ok {
				name = n
			} else {
				break
			}
		}
	}

	var zero T
	if !threadLocal.Ex(true) {
		defer threadLocal.Remove()
	}

	// checked
	value := threadLocal.Load()
	if !value.push(name) {
		// TODO - 未处理多例的情况
		return zero, warpError(fmt.Errorf("acircular dependency occurs:\n%s", join(value.g, name)))
	}

	if name == "" {
		t, err = do.Invoke[T](container.inject)
	} else {
		t, err = do.InvokeNamed[T](container.inject, name)
	}

	if err != nil {
		return
	}

	// proxy
	if px, pxErr := proxy.New[T](t); pxErr == nil {
		t = px
	}
	return
}

func ListInvokeAs[T any](container *Container) (re []T) {
	services := container.inject.ListProvidedServices()
	for _, ser := range services {
		t, err := do.InvokeNamed[T](container.inject, ser.Service)
		if err == nil {
			re = append(re, t)
		}
	}
	return
}

func warpError(err error) error {
	if err != nil {
		count := -1
		frame := runtime.CallerFrame(func(fe run.Frame) (ok bool) {
			if count > -1 {
				count++
				return count >= 2 // 第x个栈
			}

			if i := len(fe.File); fe.File[i-12:] == "container.go" &&
				fe.Function == "github.com/iocgo/sdk.warpError" {
				count = 0 // 开始计数
			}
			return
		})

		if frame != nil {
			tr := strings.Split(frame.Function, ".")
			err = errors.Join(err, fmt.Errorf(`in %s # %s:%d`, frame.File, tr[:len(tr)-1], frame.Line))
		}
	}

	return err
}

func join(slice []string, n string) (str string) {
	idx := -1
	sliceL := len(slice)
	for i, it := range slice {
		if idx == -1 && it == n {
			idx = i
		}

		switch i {
		case idx:
			str += "╭- " + it + "\n"
		case sliceL - 1:
			str += "╰> " + it + "\n"
		default:
			if idx == -1 {
				str += "   " + it + "\n"
			} else {
				str += "|  " + it + "\n"
			}
		}
	}
	return
}

func elseOf[T any](condition bool, a1, a2 T) T {
	if condition {
		return a1
	} else {
		return a2
	}
}
