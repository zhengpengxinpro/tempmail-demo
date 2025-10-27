package pool

import (
	"context"
	"sync"
)

// WorkerPool 协程池
//
// 用于限制并发协程数量，避免创建过多协程导致资源耗尽
type WorkerPool struct {
	maxWorkers int
	taskQueue  chan func()
	wg         sync.WaitGroup
}

// NewWorkerPool 创建协程池
//
// 参数:
//   - maxWorkers: 最大协程数
//   - queueSize: 任务队列大小
func NewWorkerPool(maxWorkers, queueSize int) *WorkerPool {
	pool := &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan func(), queueSize),
	}
	
	return pool
}

// Start 启动协程池
func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

// Submit 提交任务
//
// 如果队列已满，会阻塞直到有空位
func (p *WorkerPool) Submit(task func()) {
	p.taskQueue <- task
}

// TrySubmit 尝试提交任务
//
// 如果队列已满，立即返回 false
func (p *WorkerPool) TrySubmit(task func()) bool {
	select {
	case p.taskQueue <- task:
		return true
	default:
		return false
	}
}

// Stop 停止协程池
func (p *WorkerPool) Stop() {
	close(p.taskQueue)
	p.wg.Wait()
}

// worker 工作协程
func (p *WorkerPool) worker(ctx context.Context) {
	defer p.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			
			// 执行任务（捕获 panic）
			func() {
				defer func() {
					if r := recover(); r != nil {
						// 记录错误
					}
				}()
				task()
			}()
		}
	}
}
