package sdk

import "fmt"

func Assert[T any](t interface{}) T {
	obj, err := AssertToError[T](t)
	if err != nil {
		panic(err)
	}
	return obj
}

func AssertToError[T any](t interface{}) (T, error) {
	obj, ok := t.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("not T: %s", tn[T]())
	}
	return obj, nil
}

func tn[T any]() string {
	var t T

	// struct
	name := fmt.Sprintf("%T", t)
	if name != "<nil>" {
		return name
	}

	// interface
	return fmt.Sprintf("%T", new(T))
}
