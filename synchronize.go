package dynamic_goroutine_pool

import (
	"sync"
	"sync/atomic"
)

type synchronizeUint64 uint64

func (n *synchronizeUint64) Value() uint64 {
	return atomic.LoadUint64((*uint64)(n))
}

func (n *synchronizeUint64) Store(i uint64) {
	atomic.StoreUint64((*uint64)(n), i)
}

func (n *synchronizeUint64) Add(i uint64) {
	atomic.AddUint64((*uint64)(n), i)
}

type synchronizeBool struct {
	bool
	sync.RWMutex
}

func (b *synchronizeBool) Value() bool {
	b.RLock()
	defer b.RUnlock()

	return b.bool
}

func (b *synchronizeBool) Store(v bool) {
	b.Lock()
	defer b.Unlock()

	b.bool = v
}

func (b *synchronizeBool) Toggle() bool {
	b.Lock()
	defer b.Unlock()

	b.bool = !b.bool
	return b.bool
}

type synchronizeCapacity struct {
	synchronizeUint64
	eventChan chan struct{}
}

func newCapacity(value uint64) (capacity synchronizeCapacity) {
	capacity = synchronizeCapacity{
		synchronizeUint64: synchronizeUint64(value),
		eventChan:         make(chan struct{}),
	}
	go func() { capacity.eventChan <- struct{}{} }()
	return
}

func (c *synchronizeCapacity) Store(i uint64) {
	// c.synchronizeUint64.Store(i)
	atomic.StoreUint64((*uint64)(&c.synchronizeUint64), i)
	c.eventChan <- struct{}{}
}

func (c *synchronizeCapacity) Add(i uint64) {
	// c.synchronizeUint64.Add(i)
	atomic.AddUint64((*uint64)(&c.synchronizeUint64), i)
	c.eventChan <- struct{}{}
}
