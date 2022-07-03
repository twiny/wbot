package queue

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/twiny/carbon"
	"github.com/twiny/wbot"
)

// Queue
type Queue struct {
	prefix string
	mu     *sync.Mutex
	list   []string
	c      *carbon.Cache
}

// NewBadgerDBQueue
func NewBadgerDBQueue(c *carbon.Cache) (*Queue, error) {
	return &Queue{
		prefix: "queue_",
		mu:     &sync.Mutex{},
		list:   []string{},
		c:      c,
	}, nil
}

// Add
func (q *Queue) Add(req wbot.Request) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}

	//
	key := strings.Join([]string{
		q.prefix,
		hex.EncodeToString(buf.Bytes()),
	}, "_")

	// add to list
	q.mu.Lock()
	q.list = append(q.list, key)
	q.mu.Unlock()

	// encode

	return q.c.Set(key, buf.Bytes(), -1)
}

// Pop
func (q *Queue) Pop() (wbot.Request, error) {
	if len(q.list) == 0 {
		return wbot.Request{}, fmt.Errorf("queue is empty")
	}
	// get from list
	q.mu.Lock()
	key := q.list[0]
	q.list = q.list[1:]
	q.mu.Unlock()

	// get from db
	b, err := q.c.Get(key)
	if b == nil || err != nil {
		return wbot.Request{}, err
	}

	// decode
	var req wbot.Request
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&req); err != nil {
		return wbot.Request{}, err
	}

	// remove from db
	if err := q.c.Del(key); err != nil {
		return wbot.Request{}, err
	}

	return req, nil
}

// Next
func (q *Queue) Next() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.list) != 0
}

// Close
func (q *Queue) Close() error {
	q.c.Close()
	return nil
}
