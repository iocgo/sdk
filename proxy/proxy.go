package proxy

import "fmt"

var (
	constructorMap = make(map[string]func(any) any)
)

func Reg[T any](constructor func(T) T) {
	n := generateServiceName[T]()
	constructorMap[n] = func(obj any) any {
		return constructor(obj.(T))
	}
}

func New[T any](t T) (T, error) {
	n := generateServiceName[T]()
	if constructor, ok := constructorMap[n]; ok {
		return constructor(t).(T), nil
	}

	var zero T
	return zero, fmt.Errorf("no constructor found for %s", n)
}

func generateServiceName[T any]() string {
	var t T

	// struct
	name := fmt.Sprintf("%T", t)
	if name != "<nil>" {
		return name
	}

	// interface
	return fmt.Sprintf("%T", new(T))
}

func generateInstanceName(t any) string {
	return fmt.Sprintf("%T", t)
}
