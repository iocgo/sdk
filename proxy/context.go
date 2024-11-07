package proxy

type Context[T any] struct {
	In,
	Out []any
	Name        string
	PackageName string
	Receiver    T
	Do          func()
}
