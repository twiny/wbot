package crawler

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/twiny/flare"
	"github.com/twiny/wbot"
	"github.com/twiny/wbot/plugin/fetcher"
	"github.com/twiny/wbot/plugin/monitor"
	"github.com/twiny/wbot/plugin/queue"
	"github.com/twiny/wbot/plugin/store"
)

const (
	crawlRunning = 0
	crawlStopped = 1
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

		status int32
		flare  flare.Notifier

		ctx  context.Context
		stop context.CancelFunc
	}
)

func New(opts ...Option) *Crawler {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	c := &Crawler{
		wg:  new(sync.WaitGroup),
		cfg: newConfig(-1, nil, nil, nil),

		fetcher: fetcher.NewHTTPClient(),
		store:   store.NewInMemoryStore(),
		queue:   queue.NewInMemoryQueue(2048),
		metrics: monitor.NewmetricsMonitor(),

		filter:  newFilter(),
		limiter: newRateLimiter(),
		robot:   newRobotManager(false),

		stream: make(chan *wbot.Response, 1024),
		errors: make(chan error, 1024),

		status: crawlStopped,
		flare:  flare.New(),

		ctx:  ctx,
		stop: stop,
	}

	for _, opt := range opts {
		opt(c)
	}

	// this routine waits for quit signal
	go func() {
		<-c.ctx.Done()
		fmt.Println("Crawler is shutting down")

		c.flare.Cancel()

		c.queue.Close()
		c.store.Close()
		c.fetcher.Close()

		c.wg.Wait()
		close(c.stream)
		close(c.errors)
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
		return fmt.Errorf("invalid links: %v", errs)
	}

	if len(targets) == 0 {
		return fmt.Errorf("no valid links")
	}

	for _, target := range targets {
		c.start(target)
	}

	c.status = crawlRunning

	c.wg.Add(c.cfg.parallel)
	for i := 0; i < c.cfg.parallel; i++ {
		go c.crawler(i)
	}

	c.wg.Wait()
	return nil
}
func (c *Crawler) OnReponse(fn func(*wbot.Response)) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-c.flare.Done():
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
			case <-c.ctx.Done():
				return
			case <-c.flare.Done():
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
	c.stop()
}

func (c *Crawler) start(target *wbot.ParsedURL) {
	param := &wbot.Param{
		MaxBodySize: c.cfg.maxBodySize,
		UserAgent:   c.cfg.userAgents.Next(),
		Timeout:     c.cfg.timeout,
	}

	if c.cfg.proxies != nil {
		param.Proxy = c.cfg.proxies.Next()
	}

	req := &wbot.Request{
		Target: target,
		Param:  param,
		Depth:  0,
	}

	if err := c.queue.Push(c.ctx, req); err != nil {
		c.errors <- fmt.Errorf("push: %w", err)
		return
	}
}
func (c *Crawler) crawler(id int) {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if atomic.LoadInt32(&c.status) == crawlStopped && c.queue.Len() == 0 {
				c.flare.Cancel()
				return
			}

			req, err := c.queue.Pop(c.ctx)
			if err != nil {
				c.errors <- fmt.Errorf("pop: %w", err)
				continue
			}

			// if the next response will exceed the max depth,
			// we signal the crawler to stop
			if atomic.LoadInt32(&req.Depth) > c.cfg.maxDepth-1 {
				atomic.StoreInt32(&c.status, crawlStopped)
			}

			c.limiter.wait(req.Target)

			resp, err := c.fetcher.Fetch(c.ctx, req)
			if err != nil {
				c.errors <- fmt.Errorf("fetch: %w", err)
				continue
			}

			c.stream <- resp

			atomic.AddInt32(&req.Depth, 1)

			if atomic.LoadInt32(&req.Depth) > c.cfg.maxDepth {
				continue
			}

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

				if visited, err := c.store.HasVisited(c.ctx, target); visited {
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

				if err := c.queue.Push(c.ctx, nextReq); err != nil {
					c.errors <- fmt.Errorf("push: %w", err)
					continue
				}
			}
		}
	}
}
