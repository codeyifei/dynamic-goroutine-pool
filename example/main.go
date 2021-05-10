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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		if err := p.RunWithContext(ctx); err != nil {
			p.Close()
			fmt.Println(err)
			// panic(err)
		}
	}()
	// p.Run()

	taskNum := 10000
	go func() {
		var err error
		for i := 0; i < taskNum; i++ {
			if err = p.AddTaskHandler(func(i int, n *uint64) pool.HandlerFunc {
				return func() error {
					time.Sleep(time.Millisecond * 50)
					atomic.AddUint64(n, 1)
					return nil
				}
			}(i, &n)); err != nil {
				panic(err)
			}
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
				"\rcurrent goroutine quantity: %d, max current goroutine quantity: %d, process: %d / %d\033[K",
				goroutineQuantity,
				maxGoroutineQuantity,
				atomic.LoadUint64(&n),
				taskNum,
			)
		case <-closeChan:
			fmt.Printf("\nprocess: %d / %d\n", atomic.LoadUint64(&n), taskNum)
			return
		}
	}
}
