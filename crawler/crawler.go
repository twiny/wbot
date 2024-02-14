package crawler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/twiny/wbot"
	"github.com/twiny/wbot/plugin/fetcher"
	"github.com/twiny/wbot/plugin/monitor"
	"github.com/twiny/wbot/plugin/queue"
	"github.com/twiny/wbot/plugin/store"

	"github.com/twiny/flare"
)

const (
	crawlInProgress int32 = iota
	crawlFinished
)

type (
	Crawler struct {
		wg  *sync.WaitGroup
		cfg *config

		fetcher wbot.Fetcher
		store   wbot.Store
		queue   wbot.Queue
		metrics wbot.MetricsMonitor

		filter  *filter
		limiter *rateLimiter
		robot   *robotManager

		stream chan *wbot.Response
		errors chan error

		once   *sync.Once
		status int32
		quit   flare.Notifier
	}
)

func New(opts ...Option) *Crawler {
	c := &Crawler{
		wg:  new(sync.WaitGroup),
		cfg: newConfig(-1, nil, nil, nil),

		fetcher: fetcher.NewHTTPClient(),
		store:   store.NewInMemoryStore(),
		queue:   queue.NewInMemoryQueue(),
		metrics: monitor.NewmetricsMonitor(),

		filter:  newFilter(),
		limiter: newRateLimiter(),
		robot:   newRobotManager(false),

		stream: make(chan *wbot.Response, 1024),
		errors: make(chan error, 1024),

		once:   new(sync.Once),
		status: crawlFinished,
		quit:   flare.New(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// this routine waits for quit signal
	go func() {
		<-c.quit.Done()
		fmt.Println("Crawler is shutting down")
		atomic.StoreInt32(&c.status, crawlFinished)

		close(c.stream)
		close(c.errors)
		c.wg.Wait()
		c.queue.Close()
		c.store.Close()
		c.fetcher.Close()
	}()

	return c
}

func (c *Crawler) Start(links ...string) error {
	var (
		targets []*wbot.ParsedURL
		errs    []error
	)
	for _, link := range links {
		target, err := wbot.NewURL(link)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		targets = append(targets, target)
	}

	if len(errs) > 0 {
		c.quit.Cancel()
		return fmt.Errorf("invalid targets: %v", errors.Join(errs...))
	}

	if len(targets) == 0 {
		c.quit.Cancel()
		return fmt.Errorf("no valid targets: %v", errs)
	}

	atomic.StoreInt32(&c.status, crawlInProgress)
	for _, target := range targets {
		c.wg.Add(1)
		go func(t *wbot.ParsedURL) {
			if err := c.start(t); err != nil {
				c.errors <- fmt.Errorf("start: %w", err)
			}
		}(target)
	}

	for i := 0; i < c.cfg.parallel; i++ {
		c.wg.Add(1)
		go c.crawler(i)
	}

	fmt.Println("Crawler is running")
	c.wg.Wait()
	fmt.Println("Crawler has stopped")

	return nil
}
func (c *Crawler) OnReponse(fn func(*wbot.Response)) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.quit.Done():
				return
			case resp, ok := <-c.stream:
				if ok {
					fn(resp)
				}
			}
		}
	}()
}
func (c *Crawler) OnError(fn func(err error)) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.quit.Done():
				return
			case err, ok := <-c.errors:
				if ok {
					fn(err)
				}
			}
		}
	}()
}
func (c *Crawler) Stats() map[string]any {
	return map[string]any{}
}
func (c *Crawler) Stop() {
	c.quit.Cancel()
}

func (c *Crawler) start(target *wbot.ParsedURL) error {
	defer c.wg.Done()

	param := &wbot.Param{
		MaxBodySize: c.cfg.maxBodySize,
		UserAgent:   c.cfg.userAgents.Next(),
	}

	if c.cfg.proxies != nil {
		param.Proxy = c.cfg.proxies.Next()
	}

	req := &wbot.Request{
		Target: target,
		Param:  param,
		Depth:  0,
	}

	if err := c.queue.Push(context.TODO(), req); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	fmt.Printf("Crawling %s\n", target.URL.String())

	return nil
}
func (c *Crawler) crawler(id int) {
	defer c.wg.Done()

	for {
		select {
		case <-c.quit.Done():
			fmt.Printf("worker %d is stopping\n", id)
			return
		default:
			if atomic.LoadInt32(&c.status) == crawlFinished {
				c.quit.Cancel()
				return
			}

			if c.queue.IsDone() {
				c.quit.Cancel()
				return
			}

			req, err := c.queue.Pop(context.TODO())
			if err != nil {
				c.errors <- fmt.Errorf("pop: %w", err)
				continue
			}

			c.limiter.wait(req.Target)

			resp, err := c.fetcher.Fetch(context.TODO(), req)
			if err != nil {
				c.errors <- fmt.Errorf("fetch: %w", err)
				continue
			}

			atomic.AddInt32(&req.Depth, 1)
			for _, target := range resp.NextURLs {
				if !strings.Contains(target.URL.Host, req.Target.Root) {
					// can be ignored - better add log level
					// c.errors <- fmt.Errorf("hostname check: %s", target.URL.String())
					continue
				}

				// if !c.robot.Allowed(req.Param.UserAgent, req.URL.String()) {
				// 	// todo: log
				// 	continue
				// }

				if !c.filter.allow(target) {
					// can be ignored - better add log level
					// c.errors <- fmt.Errorf("allow check: %s", target.URL.String())
					continue
				}

				if visited, err := c.store.HasVisited(context.TODO(), target); visited {
					if err != nil {
						c.errors <- fmt.Errorf("store: %w", err)
						continue
					}
					// can be ignored - better add log level
					// c.errors <- fmt.Errorf("URL %s has been visited", target.URL.String())
					continue
				}

				nextReq := &wbot.Request{
					Target: target,
					Depth:  req.Depth,
					Param:  req.Param,
				}

				if err := c.queue.Push(context.TODO(), nextReq); err != nil {
					c.errors <- fmt.Errorf("push: %w", err)
					continue
				}
			}

			if req.Depth > c.cfg.maxDepth {
				c.queue.Cancel() // todo: better way to stop the queue
				// c.quit.Cancel()
				continue
			}

			c.stream <- resp
		}
	}
}
func (c *Crawler) exit() {
	c.once.Do(func() {
		atomic.StoreInt32(&c.status, crawlFinished)
		c.queue.Cancel()
		c.queue.Close()
		c.store.Close()
		c.fetcher.Close()
		close(c.stream)
		close(c.errors)
	})
}
