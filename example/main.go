package main

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"

	pool "github.com/codeyifei/dynamic-goroutine-pool"
)

func main() {
	var n uint64

	goroutine := 50
	p := pool.New(uint64(goroutine))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 20)
	defer cancel()
	go func() {
		if err := p.RunWithContext(ctx); err != nil {
			panic(err)
		}
	}()

	taskNum := 10000
	go func() {
		for i := 0; i < taskNum; i++ {
			p.AddTaskHandler(func(i int, n *uint64) pool.HandlerFunc {
				return func() error {
					time.Sleep(time.Millisecond * 50)
					atomic.AddUint64(n, 1)
					return nil
				}
			}(i, &n))
		}
		p.CloseTask()
	}()
	changeTicker := time.NewTicker(time.Second * 2)
	printTicker := time.NewTicker(time.Millisecond)
	closeChan := p.ListenClose()
	var maxGoroutineQuantity int
	rand.Seed(time.Now().Unix())
	for {
		select {
		case <-changeTicker.C:
			p.UpdateWorkerQuantity(uint64(rand.Intn(goroutine) + goroutine))
		case <-printTicker.C:
			goroutineQuantity := runtime.NumGoroutine()
			if maxGoroutineQuantity < goroutineQuantity {
				maxGoroutineQuantity = goroutineQuantity
			}
			fmt.Printf(
				"\rcurrent goruntine quantity: %d, max current goruntine quantity: %d, amount completed: %d\033[K",
				goroutineQuantity,
				maxGoroutineQuantity,
				atomic.LoadUint64(&n),
			)
		case <-closeChan:
			return
		}
	}
}
