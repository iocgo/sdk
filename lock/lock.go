package lock

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	run "github.com/iocgo/sdk/runtime"
)

type ExpireLock struct {
	// 计数
	count int64

	// goid
	gid, // -1 非重入，0可重入 & 无状态，> 0可重入 & 有状态
	reentrantCount int64

	// 核心锁
	mutex sync.Mutex
}

func NewExpireLock(reentrant bool) *ExpireLock {
	var gid int64 = -1
	if reentrant {
		gid = 0
	}

	return &ExpireLock{
		gid:   gid,
		count: 0,
	}
}

// Lock 加锁
func (e *ExpireLock) Lock(ctx context.Context) bool {
	if ctx == nil {
		timeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		ctx = timeout
	}

	atomic.AddInt64(&e.count, 1)
	for {
		select {
		case <-ctx.Done():
			// 如果上下文超时，返回false
			atomic.AddInt64(&e.count, -1)
			return false
		default:
			// 尝试获取锁
			if e.tryLock() {
				// 如果成功获取到锁，返回true
				return true
			}
			// 如果没有获取到锁，等待一段时间后重试
			// time.Sleep(100 * time.Millisecond)

			// 让出协程使用权
			runtime.Gosched()
		}
	}
}

// Lock 解锁
func (e *ExpireLock) Unlock() {
	atomic.AddInt64(&e.count, -1)
	e.unlock()
}

func (e *ExpireLock) tryLock() (ok bool) {
	ok = e.mutex.TryLock()
	if ok { // 成功上锁
		if e.gid >= 0 { // 开启了可重入
			atomic.StoreInt64(&e.gid, run.GetCurrentGoroutineID())
		}
		e.reentrantCount++
		return
	}

	// 上锁失败，检查是否需要重入
	if e.gid >= 0 { // 开启了可重入
		gid := run.GetCurrentGoroutineID()
		if atomic.CompareAndSwapInt64(&e.gid, gid, gid) { // 允许重入
			e.reentrantCount++
			return true
		}
	}
	return
}

func (e *ExpireLock) unlock() {
	if e.gid >= 0 { // 开启了可重入
		e.reentrantCount--
		if e.reentrantCount <= 0 {
			atomic.StoreInt64(&e.gid, 0) // 设置为无状态
			e.mutex.Unlock()
		}
		return
	}

	e.mutex.Unlock()
}

// Lock 是否空闲
func (e *ExpireLock) IsIdle() bool {
	return atomic.LoadInt64(&e.count) < 1
}
