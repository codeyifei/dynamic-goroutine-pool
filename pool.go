package dynamic_goroutine_pool

import "context"

// 协程池
type Pool struct {
	capacity       synchronizeCapacity
	runningWorkers synchronizeUint64
	closeChan      chan struct{}
	taskChan       chan HandlerFunc
	errorChan      chan error
	isTaskClose    synchronizeBool
}

// 初始化协程池，capacity为协程池初始容量
// 实例代码：
// p := New(10)
// 值得注意:
// 1. 协程池中协程的数量仅受控于capacity，而不受task数量的影响。如需要通过剩余task数控制协程池的容量，请通过协程池的方法自行实现
// 2. 任务通道的缓存大小与初始化协程数一致，暂不支持自定义缓存大小
func New(capacity uint64) *Pool {
	return &Pool{
		capacity:   newCapacity(capacity),
		closeChan:  make(chan struct{}, capacity),
		taskChan:   make(chan HandlerFunc, capacity),
		errorChan:  make(chan error),
	}
}

// 向线程池添加任务
func (p *Pool) AddTask(task Task) error {
	return p.AddTaskHandler(func() error {
		return task.Handler(task.Params...)
	})
}

// 向线程池添加任务处理函数
func (p *Pool) AddTaskHandler(handlerFunc HandlerFunc) error {
	if p.isTaskClose.Value() {
		return ErrTaskChannelIsClosed
	}
	p.taskChan <- handlerFunc
	return nil
}

// 获取剩余任务数，可在外部通过获取该值动态改变协程池的容量
func (p *Pool) RemainingTaskQuantity() int {
	return len(p.taskChan)
}

// 修改协程池的容量
func (p *Pool) UpdateWorkerQuantity(quantity uint64) {
	p.capacity.Store(quantity)
	// p.updateChan <- struct{}{}
}

// 容量增加一
func (p *Pool) IncWorker() {
	p.capacity.Add(1)
	// p.updateChan <- struct{}{}
}

// 容量减少一
func (p *Pool) DecWorker() {
	p.capacity.Add(^uint64(0))
	// p.updateChan <- struct{}{}
}

// 运行协程池
func (p *Pool) Run() {
	go p.dynamic()
	// p.updateChan <- struct{}{}
}

// 使用上下文运行协程池
func (p *Pool) RunWithContext(ctx context.Context) error {
	p.Run()
	<-ctx.Done()
	return ctx.Err()
}

// 监听协程池的关闭
func (p *Pool) ListenClose() chan struct{} {
	c := make(chan struct{})
	go func() {
		for {
			if p.isTaskClose.Value() && len(p.taskChan) == 0 && p.runningWorkers.Value() == 0 {
				c <- struct{}{}
				return
			}
		}
	}()
	return c
}

// 等待协程池的关闭
func (p *Pool) Wait() {
	<-p.ListenClose()
}

// 关闭协程池，协程池需要手动进行关闭
func (p *Pool) Close() {
	p.CloseTask()
	p.UpdateWorkerQuantity(0)
}

// 关闭任务通道
func (p *Pool) CloseTask() {
	close(p.taskChan)
	p.isTaskClose.Store(true)
}

// 通过监听容量的变化实时修改当前协程数量
func (p *Pool) dynamic() {
	for {
		select {
		case <- p.capacity.eventChan:
			runningWorkers := p.runningWorkers.Value()
			capacity := p.capacity.Value()
			switch {
			case runningWorkers < capacity:
				for i := 0; i < int(capacity-runningWorkers); i++ {
					p.run()
				}
			case runningWorkers > capacity:
				for i := 0; i < int(runningWorkers-capacity); i++ {
					p.closeChan <- struct{}{}
				}
			}
		}
	}
}

// 运行一个协程
func (p *Pool) run() {
	p.runningWorkers.Add(1)
	go func() {
		defer func() { p.runningWorkers.Add(^uint64(0)) }()

		for {
			select {
			case task, ok := <-p.taskChan:
				if !ok {
					return
				}
				if err := task(); err != nil {
					p.errorChan <- err
				}
			case <-p.closeChan:
				return
			}
		}
	}()
}

// 获取错误通道
// 值得注意的是，默认错误通道为无缓存通道，为避免出现错误后阻塞协程，请及时接收并处理错误信息
func (p *Pool) Errors() chan error {
	return p.errorChan
}
