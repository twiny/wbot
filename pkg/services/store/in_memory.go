package store

import (
	"context"
	"sync"

	"github.com/twiny/wbot/pkg/api"
)

type (
	defaultInMemoryStore struct {
		mu    sync.Mutex
		table map[string]bool
	}
)

func NewInMemoryStore() api.Store {
	return &defaultInMemoryStore{
		table: make(map[string]bool),
	}
}
func (s *defaultInMemoryStore) HasVisited(ctx context.Context, link *api.ParsedURL) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.table[link.Hash]
	if !found {
		s.table[link.Hash] = true
		return false, nil
	}

	return found, nil
}
func (s *defaultInMemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	clear(s.table)
	return nil
}
