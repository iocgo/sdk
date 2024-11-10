package proxy

import "reflect"

type Context struct {
	In,
	Out []any
	Name     string
	Receiver reflect.Value
	Do       func()
}
