package store

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/twiny/carbon"
)

// Store
type Store struct {
	prefix string
	c      *carbon.Cache
}

// NewBadgerDBStore
func NewBadgerDBStore(c *carbon.Cache) (*Store, error) {
	return &Store{
		prefix: prefix,
		c:      c,
	}, nil
}

// Visited
func (s *Store) Visited(link string) bool {
	hash := sha256.Sum224([]byte(link))

	//
	key := strings.Join([]string{
		s.prefix,
		hex.EncodeToString(hash[:]),
	}, "_")

	// check if already visited
	if d, err := s.c.Get(key); d == nil || err != nil {
		// if not found save it and return false
		if err := s.c.Set(key, []byte(link), -1); err != nil {
			return false
		}

		return false
	}

	return true
}

// Close
func (s *Store) Close() error {
	s.c.Close()
	return nil
}
