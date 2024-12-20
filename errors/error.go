package errors

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

type Context struct {
	err   error
	catch func(err error) bool
	ok    bool
}

func (ctx *Context) Error() error {
	return ctx.err
}

func (ctx *Context) Throw() {
	err := recover()
	if err == nil {
		return
	}

	if ctx.err == nil {
		// panic(err)
		fmt.Printf("\npanic: %v\n\n", err)
		os.Stderr.Write(stack())
		os.Exit(2)
	}

	if ctx.ok {
		return
	}

	panic(ctx.err)
}

func stack() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

func New(catch func(err error) bool) *Context {
	return &Context{
		err:   nil,
		catch: catch,
	}
}

func panicTo(ctx *Context) {
	if ctx.err == nil {
		return
	}

	if ctx.catch != nil {
		ctx.ok = ctx.catch(ctx.err)
	}

	panic(io.EOF)
}

func Try(ctx *Context, exec func() error) {
	ctx.err = exec()
	panicTo(ctx)
}

func Try1[T any](ctx *Context, exec func() (T, error)) (t T) {
	t, ctx.err = exec()
	panicTo(ctx)
	return
}

func Try2[T, M any](ctx *Context, exec func() (T, M, error)) (t T, m M) {
	t, m, ctx.err = exec()
	panicTo(ctx)
	return
}

func Try3[T, M, E any](ctx *Context, exec func() (T, M, E, error)) (t T, m M, e E) {
	t, m, e, ctx.err = exec()
	panicTo(ctx)
	return
}

func Try4[T, M, E, D any](ctx *Context, exec func() (T, M, E, D, error)) (t T, m M, e E, d D) {
	t, m, e, d, ctx.err = exec()
	panicTo(ctx)
	return
}

func Try5[T, M, E, D, G any](ctx *Context, exec func() (T, M, E, D, G, error)) (t T, m M, e E, d D, g G) {
	t, m, e, d, g, ctx.err = exec()
	panicTo(ctx)
	return
}
