package proxy

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"
)

var (
	constructorMap = make(map[string]func(any) any)
)

func Reg[T any](constructor func(T) T) {
	n, ok := generateInterfaceName[T]()
	if !ok {
		panic("this T type is not interface: " + n)
	}
	constructorMap[n] = func(obj any) any {
		return constructor(obj.(T))
	}
}

func New[T any](t T) (T, error) {
	var zero T
	n, isInter := generateInterfaceName[T]()
	if !isInter {
		return zero, fmt.Errorf("this T type is not interface: %s" + n)
	}
	if constructor, ok := constructorMap[n]; ok {
		return constructor(t).(T), nil
	}
	return zero, fmt.Errorf("no constructor found for %s", n)
}

func generateInterfaceName[T any]() (name string, isInter bool) {
	var t T

	ox := reflect.TypeOf(t)
	if ox == nil {
		ox = reflect.TypeOf(new(T))
	}

	if ox.Kind() == reflect.Ptr {
		ox = ox.Elem()
	}

	name = ox.String()
	idx := strings.LastIndexByte(name, '.')
	name = ox.PkgPath() + name[idx:]
	isInter = ox.Kind() == reflect.Interface
	return
}

func valueType(obj interface{}) string {
	ox := reflect.ValueOf(obj)
	isPtr := ox.Kind() == reflect.Ptr
	if isPtr {
		ox = ox.Elem()
	}

	value := fmt.Sprintf("%T", ox.Interface())
	if !strings.Contains(value, "/") {
		idx := strings.LastIndex(value, ".")
		return ox.Type().PkgPath() + value[idx:]
	}

	return value
}

func Matched(regex string, obj interface{}) bool {
	if len(regex) == 0 {
		panic(errors.New("regex is empty"))
	}

	matched, err := path.Match(regex, valueType(obj))
	if err != nil {
		panic(err)
	}

	return matched
}
