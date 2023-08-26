package crawler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/twiny/wbot"
	"github.com/twiny/wbot/plugin/fetcher"
	"github.com/twiny/wbot/plugin/queue"
	"github.com/twiny/wbot/plugin/store"

	"github.com/twiny/flare"

	clog "github.com/charmbracelet/log"
)

type CrawlState int32

const (
	CrawlQuit CrawlState = iota
	CrawlFinished
	CrawlInProgress
	CrawlPaused
)

type (
	Crawler struct {
		wg sync.WaitGroup

		cfg *config

		fetcher wbot.Fetcher
		store   wbot.Store
		logger  wbot.Logger
		queue   wbot.Queue
		metrics wbot.MetricsMonitor

		filter  *filter
		limiter *rateLimiter
		robot   *robortManager

		counter int32
		stream  chan *wbot.Response
		errors  chan error

		state CrawlState
		quit  flare.Notifier

		termLog *clog.Logger
	}
)

func New(opts ...Option) *Crawler {
	logOpt := clog.Options{
		TimeFormat:      "2006-01-02 15:04:05",
		Level:           clog.DebugLevel,
		Prefix:          "[WBot]",
		ReportTimestamp: true,
	}

	c := &Crawler{
		cfg: newConfig(-1, nil, nil, nil),

		fetcher: fetcher.NewHTTPClient(),
		store:   store.NewInMemoryStore(),
		// logger:  newFileLogger(),
		queue: queue.NewInMemoryQueue(),
		// metrics: newMetricsMonitor(),

		filter:  newFilter(),
		limiter: newRateLimiter(),
		// robot:   newRobotsManager(),

		stream: make(chan *wbot.Response, 16),
		errors: make(chan error, 16),

		termLog: clog.NewWithOptions(os.Stdout, logOpt),

		quit: flare.New(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Crawler) Start(links ...string) error {
	var targets []*wbot.ParsedURL
	for _, link := range links {
		target, err := wbot.NewURL(link)
		if err != nil {
			c.errors <- fmt.Errorf("start: %w", err)
			continue
		}
		targets = append(targets, target)
	}

	if len(targets) == 0 {
		c.quit.Cancel()
		c.wg.Wait()
		close(c.stream)
		return fmt.Errorf("no links to crawl")
	}

	c.termLog.Infof("crawling %d links...", len(targets))

	for _, target := range targets {
		c.wg.Add(1)
		go c.start(target)
	}

	c.wg.Add(c.cfg.parallel)
	c.termLog.Infof("starting %d workers...", c.cfg.parallel)
	for i := 0; i < c.cfg.parallel; i++ {
		go c.crawler()
	}

	c.wg.Wait()
	return nil
}
func (c *Crawler) OnReponse(fn func(*wbot.Response) error) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			select {
			case <-c.quit.Done():
				return
			case resp := <-c.stream:
				if err := fn(resp); err != nil {
					c.errors <- err
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
			case err := <-c.errors:
				fn(err)
			}
		}
	}()
}
func (c *Crawler) Stats() map[string]any {
	return map[string]any{}
}
func (c *Crawler) Close() {
	c.termLog.Debugf("closing...")
	c.quit.Cancel()
	c.wg.Wait()
	c.store.Close()
	c.fetcher.Close()
}

func (c *Crawler) start(target *wbot.ParsedURL) {
	defer func() {
		c.wg.Done()
		atomic.AddInt32(&c.counter, 1)
	}()

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
		c.errors <- fmt.Errorf("push: %w", err)
		return
	}

}
func (c *Crawler) crawler() {
	defer c.wg.Done()

	for {
		select {
		case <-c.quit.Done():
			c.termLog.Debugf("quit")
			c.queue.Close() // must close queue before quit
			return
		default:
			func() {
				defer atomic.AddInt32(&c.counter, -1)
			}()

			req, err := c.queue.Pop(context.TODO())
			if err != nil {
				// atomic.AddInt32(&c.counter, -1)
				// c.termLog.Errorf("pop: %s", err.Error())
				continue
			}

			if req.Depth > c.cfg.maxDepth {
				// atomic.AddInt32(&c.counter, -1)

				if c.queue.Len()+atomic.LoadInt32(&c.counter) == 0 {
					c.quit.Cancel()
					fmt.Println("queue len:", c.queue.Len())
					continue
				}

				continue
			}

			c.limiter.wait(req.Target)

			resp, err := c.fetcher.Fetch(context.TODO(), req)
			if err != nil {
				// todo: log
				// atomic.AddInt32(&c.counter, -1)
				c.errors <- fmt.Errorf("fetch: %w", err)
				continue
			}

			atomic.AddInt32(&req.Depth, 1)
			for _, target := range resp.NextURLs {
				if !strings.Contains(target.URL.Host, req.Target.Root) {
					c.errors <- fmt.Errorf("hostname check: %w", err)
					continue
				}

				// if !c.robot.Allowed(req.Param.UserAgent, req.URL.String()) {
				// 	// todo: log
				// 	continue
				// }

				if !c.filter.allow(target) {
					c.errors <- fmt.Errorf("allow check: %w", err)
					continue
				}

				if visited, err := c.store.HasVisited(context.TODO(), target); visited {
					if err != nil {
						c.errors <- fmt.Errorf("has visited 1: %w", err)
						continue
					}
					c.errors <- fmt.Errorf("has visited 2: %w", err)
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

				atomic.AddInt32(&c.counter, 1)
			}

			// if c.log != nil {
			// 	rep := newReport(resp, nil)
			// 	c.log.Send(rep)
			// }

			// stream
			c.stream <- resp
			// atomic.AddInt32(&c.counter, -1)

			c.termLog.Infof("crawled: %s, depth: %d, counter: %d, queue: %d", first64Chars(req.Target.URL.String()), req.Depth, c.counter, c.queue.Len())

			if c.queue.Len()+atomic.LoadInt32(&c.counter) == 0 {
				c.queue.Close() // must close queue before quit
				continue
			}
		}
	}
}

func first64Chars(s string) string {
	if len(s) <= 64 {
		return s
	}

	runes := []rune(s)
	if len(runes) <= 64 {
		return s
	}

	return string(runes[:64])
}
