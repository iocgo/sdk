package runtime

import (
	"fmt"
	"runtime"
	"sync"
)

var (
	littleBuf = sync.Pool{
		New: func() any { bytes := make([]byte, 64); return &bytes },
	}
)

func GetCurrentGoroutineID() (id int64) {
	bp := littleBuf.Get().(*[]byte)
	defer littleBuf.Put(bp)

	b := *bp
	b = b[:runtime.Stack(b, false)]
	_, err := fmt.Sscanf(string(b), "goroutine %d [", &id)
	if err != nil {
		panic(err)
	}
	return
}
