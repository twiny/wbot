package queue

import (
	"context"
	"fmt"
	"sync"

	"github.com/twiny/wbot"
)

var (
	ErrQueueClosed = fmt.Errorf("queue is closed")
)

type (
	defaultInMemoryQueue struct {
		mu     *sync.RWMutex
		list   []*wbot.Request
		cond   *sync.Cond
		closed bool
	}
)

func NewInMemoryQueue() wbot.Queue {
	queue := &defaultInMemoryQueue{
		mu:   &sync.RWMutex{},
		list: make([]*wbot.Request, 0, 4096),
	}
	queue.cond = sync.NewCond(queue.mu)
	return queue
}
func (q *defaultInMemoryQueue) Push(ctx context.Context, req *wbot.Request) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	q.list = append(q.list, req)
	q.cond.Broadcast()

	return nil
}
func (q *defaultInMemoryQueue) Pop(ctx context.Context) (*wbot.Request, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	switch {
	case q.closed && len(q.list) == 0:
		return nil, ErrQueueClosed
	case len(q.list) == 0 && !q.closed:
		q.cond.Wait()
	}

	req := q.list[0]
	q.list = q.list[1:]
	return req, nil
}
func (q *defaultInMemoryQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.list)
}
func (q *defaultInMemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.closed = true
	q.cond.Broadcast()

	return nil
}
