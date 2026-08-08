package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// TaskFunc is a function that processes a task.
type TaskFunc func(payload map[string]interface{}) error

// Task represents a unit of work.
type Task struct {
	ID      string
	Type    string
	Payload map[string]interface{}
	Attempts int
	MaxRetry int
}

// JobQueue is an in-memory async task queue with retry support.
type JobQueue struct {
	mu        sync.RWMutex
	handlers  map[string]TaskFunc
	queue     chan *Task
	stopCh    chan struct{}
	wg        sync.WaitGroup
	workers   int
	stats     QueueStats
}

// QueueStats tracks queue statistics.
type QueueStats struct {
	Enqueued  int64 `json:"enqueued"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
	Retried   int64 `json:"retried"`
	Pending   int64 `json:"pending"`
}

// New creates a new JobQueue with the specified number of workers.
func New(workers int) *JobQueue {
	if workers <= 0 {
		workers = 4
	}
	q := &JobQueue{
		handlers: make(map[string]TaskFunc),
		queue:    make(chan *Task, 1000),
		stopCh:   make(chan struct{}),
		workers:  workers,
	}
	q.start()
	return q
}

// Register associates a task type with a handler function.
func (q *JobQueue) Register(taskType string, fn TaskFunc) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[taskType] = fn
}

// Enqueue adds a task to the queue. Returns an error if the queue is full.
func (q *JobQueue) Enqueue(taskType string, payload map[string]interface{}, maxRetry int) error {
	task := &Task{
		ID:       fmt.Sprintf("%s-%d", taskType, time.Now().UnixNano()),
		Type:     taskType,
		Payload:  payload,
		MaxRetry: maxRetry,
	}
	select {
	case q.queue <- task:
		atomic.AddInt64(&q.stats.Enqueued, 1)
		atomic.AddInt64(&q.stats.Pending, 1)
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// Stats returns current queue statistics.
func (q *JobQueue) Stats() QueueStats {
	return QueueStats{
		Enqueued:  atomic.LoadInt64(&q.stats.Enqueued),
		Completed: atomic.LoadInt64(&q.stats.Completed),
		Failed:    atomic.LoadInt64(&q.stats.Failed),
		Retried:   atomic.LoadInt64(&q.stats.Retried),
		Pending:   atomic.LoadInt64(&q.stats.Pending),
	}
}

// Stop gracefully shuts down the queue, waiting for pending tasks.
func (q *JobQueue) Stop() {
	close(q.stopCh)
	close(q.queue)
	q.wg.Wait()
}

// start launches worker goroutines.
func (q *JobQueue) start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// worker processes tasks from the queue.
func (q *JobQueue) worker(id int) {
	defer q.wg.Done()
	for {
		select {
		case <-q.stopCh:
			return
		case task, ok := <-q.queue:
			if !ok {
				return
			}
			q.processTask(task)
		}
	}
}

// processTask executes a task with retry logic.
func (q *JobQueue) processTask(task *Task) {
	q.mu.RLock()
	handler, ok := q.handlers[task.Type]
	q.mu.RUnlock()

	if !ok {
		log.Printf("[JobQueue] no handler for task type: %s", task.Type)
		atomic.AddInt64(&q.stats.Failed, 1)
		atomic.AddInt64(&q.stats.Pending, -1)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = ctx // available for handler use if needed

	err := handler(task.Payload)
	if err != nil {
		task.Attempts++
		if task.Attempts < task.MaxRetry {
			atomic.AddInt64(&q.stats.Retried, 1)
			// Exponential backoff: 2s, 4s, 8s...
			backoff := time.Duration(1<<task.Attempts) * time.Second
			time.Sleep(backoff)
			// Re-enqueue
			select {
			case q.queue <- task:
			default:
				log.Printf("[JobQueue] queue full, dropping retry for task %s", task.ID)
				atomic.AddInt64(&q.stats.Failed, 1)
				atomic.AddInt64(&q.stats.Pending, -1)
			}
			return
		}
		log.Printf("[JobQueue] task %s failed after %d attempts: %v", task.ID, task.Attempts, err)
		atomic.AddInt64(&q.stats.Failed, 1)
		atomic.AddInt64(&q.stats.Pending, -1)
		return
	}

	atomic.AddInt64(&q.stats.Completed, 1)
	atomic.AddInt64(&q.stats.Pending, -1)
}

// Global job queue instance
var defaultQueue *JobQueue

// Init initializes the global job queue.
func Init(workers int) {
	if defaultQueue == nil {
		defaultQueue = New(workers)
	}
}

// Default returns the global job queue instance.
func Default() *JobQueue {
	if defaultQueue == nil {
		Init(4)
	}
	return defaultQueue
}
