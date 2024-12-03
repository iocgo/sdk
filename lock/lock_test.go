package lock

import (
	"context"
	"fmt"
	"github.com/iocgo/sdk/runtime"
	"testing"
	"time"
)

func elseOf(condition bool, a1, a2 string) string {
	if condition {
		return a1
	}
	return a2
}

func TestExpireLock(t *testing.T) {
	expireLock := NewExpireLock(true)
	gid := runtime.GetCurrentGoroutineID()
	fmt.Printf("就绪[main - %d]\n", gid)

	time.Sleep(time.Second)
	timeout, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	fmt.Printf("%s获取到锁1[main - %d]\n", elseOf(expireLock.Lock(timeout), "", "没有"), gid)

	timeout, cancel = context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	fmt.Printf("%s获取到锁2[main - %d]\n", elseOf(expireLock.Lock(timeout), "", "没有"), gid)

	go func() {
		id := runtime.GetCurrentGoroutineID()
		timeout1, cancel1 := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel1()
		ok := expireLock.Lock(timeout1)
		fmt.Printf("%s获取到锁[go routine - %d]\n", elseOf(ok, "", "没有"), id)
		if ok {
			time.Sleep(3 * time.Second)
			expireLock.Unlock()
			fmt.Printf("释放锁[go routine - %d]\n", id)
		}
	}()

	time.Sleep(5 * time.Second)

	expireLock.Unlock()
	fmt.Printf("解锁1[main - %d]: %d\n", gid, expireLock.reentrantCount)
	expireLock.Unlock()
	fmt.Printf("解锁2[main - %d]: %d\n", gid, expireLock.reentrantCount)
}
