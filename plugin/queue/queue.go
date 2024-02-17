package queue

import (
	"context"
	"fmt"
	"sync"

	"github.com/twiny/wbot"
)

/*
read first page add requests to queue
if request depth is exceeded return
*/
type defaultInMemoryQueue struct {
	mu   *sync.RWMutex
	list []*wbot.Request
}

func NewInMemoryQueue(size int) wbot.Queue {
	q := &defaultInMemoryQueue{
		mu:   new(sync.RWMutex),
		list: make([]*wbot.Request, 0, size),
	}

	return q
}

func (q *defaultInMemoryQueue) Push(ctx context.Context, req *wbot.Request) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.list = append(q.list, req)

	return nil
}
func (q *defaultInMemoryQueue) Pop(ctx context.Context) (*wbot.Request, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.list) == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	req := q.list[0]
	q.list = q.list[1:]

	return req, nil
}
func (q *defaultInMemoryQueue) Len() int32 {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return int32(len(q.list))
}
func (q *defaultInMemoryQueue) Close() error {
	clear(q.list)
	return nil
}
