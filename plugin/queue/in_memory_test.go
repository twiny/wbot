package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/twiny/wbot"
)

func TestInMemoryPush(t *testing.T) {

}

func TestInMemoryPop(t *testing.T) {

}

// go test -benchmem -v -count=1 -run=^$ -bench ^BenchmarkInMemoryPush$ github.com/twiny/wbot/plugin/queue -tags=integration,unit
func BenchmarkInMemoryPush(b *testing.B) {
	queue := NewInMemoryQueue()
	defer queue.Close()

	b.ResetTimer() // Reset the timer to ignore the setup time

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			if err := queue.Push(context.TODO(), &wbot.Request{
				ID: fmt.Sprintf("%d", j),
			}); err != nil {
				b.Error(err)
			}
		}(i)
	}
	wg.Wait()
}

func BenchmarkInMemoryPop(b *testing.B) {

}
