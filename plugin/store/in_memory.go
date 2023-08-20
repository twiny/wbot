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

	hash, err := wbot.HashLink(link)
	if err != nil {
		return false, err
	}

	_, found := s.table[hash]
	if !found {
		s.table[hash] = true
		return false, nil
	}

	return found, nil
}
func (s *defaultInMemoryStore) Close() error {
	return nil
}
