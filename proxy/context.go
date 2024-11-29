package proxy

import "reflect"

type Context struct {
	In,
	Out []any
	Method   string
	Receiver reflect.Value
	Do       func()
}
