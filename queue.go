package wbot

import (
	"fmt"
	"sync"
)

// Queue
type Queue interface {
	Add(req Request) error
	Pop() (Request, error)
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
func (q *queue[T]) Add(item T) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.q = append(q.q, item)

	return nil
}

// Pop
func (q *queue[T]) Pop() (T, error) {
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
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.q) > 0
}

// Close
func (q *queue[T]) Close() error {
	q.q = nil
	return nil
}
