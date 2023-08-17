package store

import (
	"context"
	"sync"

	"github.com/twiny/wbot"
)

type (
	defaultInMemoryStore struct {
		mu    sync.RWMutex
		table map[string]bool
	}
)

func NewInMemoryStore() wbot.Store {
	return &defaultInMemoryStore{
		table: make(map[string]bool),
	}
}
func (s *defaultInMemoryStore) HasVisited(ctx context.Context, link string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.table[link], nil
}
func (s *defaultInMemoryStore) Close() error {
	return nil
}
