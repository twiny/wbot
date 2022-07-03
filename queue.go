package wbot

import (
	"fmt"
	"sync"
)

// Queue
type Queue interface {
	Enqueue(req Request) error
	Dequeue() (Request, error)
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

// Enqueue
func (q *queue[T]) Enqueue(item T) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.q = append(q.q, item)

	return nil
}

// Dequeue
func (q *queue[T]) Dequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.q) == 0 {
		var t T
		return t, fmt.Errorf("queue is empty")
	}
	r := q.q[0]
	q.q = q.q[1:]
	return r, nil
}

// Next
func (q *queue[T]) Next() bool {
	return len(q.q) != 0
}

// Close
func (q *queue[T]) Close() error {
	q.q = nil
	return nil
}
