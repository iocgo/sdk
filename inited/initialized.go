package inited

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	inits = make([]func(args ...interface{}), 0)
	exits = make([]func(args ...interface{}), 0)
)

func AddInitialized(apply func(args ...interface{})) { inits = append(inits, apply) }
func AddExited(apply func(args ...interface{}))      { exits = append(exits, apply) }
func Initialized(block bool, args ...interface{}) {
	for _, apply := range inits {
		apply(args...)
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	if block {
		<-osSignal
		for _, apply := range exits {
			apply(args...)
		}
		os.Exit(0)
		return
	}

	go func(ch chan os.Signal) {
		<-ch
		for _, apply := range exits {
			apply(args...)
		}
		os.Exit(0)
	}(osSignal)
}
