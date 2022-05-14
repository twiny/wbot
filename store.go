package wbot

import "sync"

// Store
type Store interface {
	Visited(link string) bool
	Close()
}

// Default Store

//

// Store
type store[T comparable] struct {
	mu      *sync.Mutex
	visited map[T]bool
}

// NewStore
func defaultStore[T comparable]() *store[T] {
	return &store[T]{
		mu:      &sync.Mutex{},
		visited: make(map[T]bool),
	}
}

// Visited
func (s *store[T]) Visited(k T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.visited[k]

	// add if not visited
	if !ok {
		s.visited[k] = true
	}

	return ok
}

// Close
func (s *store[T]) Close() {
	// nothing to do
}
