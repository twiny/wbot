package wbot

import "sync"

// Queue
type Queue interface {
	Add(req *Request)
	Pop() *Request
	Next() bool
	Close() error
}

// Default Queue

// Queue
type queue[T any] struct {
	mu *sync.Mutex
	q  []T
}

// NewQueue
func defaultQueue[T any]() *queue[T] {
	return &queue[T]{
		mu: &sync.Mutex{},
		q:  make([]T, 0),
	}
}

// Add
func (q *queue[T]) Add(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.q = append(q.q, item)
}

// Pop
func (q *queue[T]) Pop() T {
	q.mu.Lock()
	defer q.mu.Unlock()
	// if len(q.q) == 0 {
	// 	return nil
	// }
	r := q.q[0]
	q.q = q.q[1:]
	return r
}

// Next
func (q *queue[T]) Next() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.q) > 0
}

// Close
func (q *queue[T]) Close() error {
	q.q = nil
	return nil
}
