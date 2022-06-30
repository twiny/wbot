package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/twiny/wbot"
	"go.etcd.io/bbolt"
)

var (
	prefix = "store_"
)

// BBoltStore
type BBoltStore struct {
	prefix string
	db     *bbolt.DB
}

// NewBBoltStore
func NewBBoltStore(db *bbolt.DB) (wbot.Store, error) {
	// create bucket for store
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(prefix))
		return err
	}); err != nil {
		return nil, err
	}

	return &BBoltStore{
		prefix: prefix,
		db:     db,
	}, nil
}

// Visited
func (bs *BBoltStore) Visited(link string) bool {
	hash := sha256.Sum224([]byte(link))

	//
	key := strings.Join([]string{
		bs.prefix,
		hex.EncodeToString(hash[:]),
	}, "")

	return bs.db.Update(func(tx *bbolt.Tx) error {
		bu := tx.Bucket([]byte(prefix))

		if d := bu.Get([]byte(key)); d == nil {
			// if not found save it and return false
			return bu.Put([]byte(key), []byte(link))
		}

		return fmt.Errorf("visited")
	}) == nil
}

// Close
func (bs *BBoltStore) Close() error {
	return bs.db.Close()
}
