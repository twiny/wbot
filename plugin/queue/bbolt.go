package queue

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/twiny/wbot"
	"go.etcd.io/bbolt"
)

var prefix = "queue_"

// BBoltQueue
type BBoltQueue struct {
	prefix string
	mu     *sync.Mutex
	list   []string
	db     *bbolt.DB
}

// NewBBoltQueue
func NewBBoltQueue(db *bbolt.DB) (wbot.Queue, error) {
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(prefix))
		return err
	}); err != nil {
		return nil, err
	}
	return &BBoltQueue{
		prefix: prefix,
		mu:     &sync.Mutex{},
		list:   []string{},
		db:     db,
	}, nil
}

// Add
func (bq *BBoltQueue) Add(req wbot.Request) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}

	//
	key := strings.Join([]string{
		bq.prefix,
		hex.EncodeToString(buf.Bytes()),
	}, "")

	// add to list
	bq.mu.Lock()
	bq.list = append(bq.list, key)
	bq.mu.Unlock()

	return bq.db.Update(func(tx *bbolt.Tx) error {
		bu := tx.Bucket([]byte(prefix))

		return bu.Put([]byte(key), buf.Bytes())
	})
}

// Pop
func (bq *BBoltQueue) Pop() (wbot.Request, error) {
	if len(bq.list) == 0 {
		return wbot.Request{}, fmt.Errorf("queue is empty")
	}
	// get from list
	bq.mu.Lock()
	key := bq.list[0]
	bq.list = bq.list[1:]
	bq.mu.Unlock()

	// get from db
	var req wbot.Request
	if err := bq.db.View(func(tx *bbolt.Tx) error {
		bu := tx.Bucket([]byte(prefix))

		if b := bu.Get([]byte(key)); b != nil {
			if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&req); err != nil {
				return err
			}
			return nil
		}

		return fmt.Errorf("not found")
	}); err != nil {
		return wbot.Request{}, err
	}

	// remove from db
	if err := bq.db.Update(func(tx *bbolt.Tx) error {
		bu := tx.Bucket([]byte(prefix))

		return bu.Delete([]byte(key))
	}); err != nil {
		return wbot.Request{}, err
	}

	return req, nil
}

// Next
func (bq *BBoltQueue) Next() bool {
	return len(bq.list) != 0
}

// Close
func (bq *BBoltQueue) Close() error {
	return bq.db.Close()
}
