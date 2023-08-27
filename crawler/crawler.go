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

		stream chan *wbot.Response
		errors chan error

		state   CrawlState
		workers int32
		quit    flare.Notifier
		close   flare.Notifier

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

		quit:  flare.New(),
		close: flare.New(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// this routine waits for quit signal
	go func() {
		<-c.quit.Done()
		close(c.stream)
		close(c.errors)
		c.exit()
	}()

	return c
}

func (c *Crawler) Start(links ...string) {
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
		c.exit()
		c.termLog.Errorf("no valid links to crawl")
		return
	}

	c.termLog.Infof("crawling %d links...", len(targets))

	for _, target := range targets {
		c.wg.Add(1)
		go c.start(target)
	}

	c.wg.Add(c.cfg.parallel)
	c.termLog.Infof("starting %d workers...", c.cfg.parallel)
	for i := 0; i < c.cfg.parallel; i++ {
		go c.crawler(i)
	}

	c.wg.Wait()
}
func (c *Crawler) OnReponse(fn func(*wbot.Response)) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case resp, ok := <-c.stream:
				if !ok {
					return
				}
				fn(resp)
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
			case err, ok := <-c.errors:
				if !ok {
					return
				}
				fn(err)
			}
		}
	}()
}
func (c *Crawler) Stats() map[string]any {
	return map[string]any{}
}
func (c *Crawler) Stop() {
	c.close.Cancel()
	c.wg.Wait()
	c.exit()
}

func (c *Crawler) start(target *wbot.ParsedURL) {
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
		c.errors <- fmt.Errorf("push: %w", err)
		return
	}
}
func (c *Crawler) crawler(id int) {
	signal := flare.New()

	defer c.wg.Done()

	for {
		select {
		case <-signal.Done():
			if atomic.AddInt32(&c.workers, 1) == int32(c.cfg.parallel) {
				c.termLog.Debugf("all workers done")
				c.quit.Cancel()
			}
			return
		case <-c.close.Done():
			if atomic.AddInt32(&c.workers, 1) == int32(c.cfg.parallel) {
				c.termLog.Debugf("all workers done")
				c.quit.Cancel()
			}
			return
		default:
			if c.queue.IsDone() {
				signal.Cancel()
				continue
			}

			req, err := c.queue.Pop(context.TODO())
			if err != nil {
				c.errors <- fmt.Errorf("pop: %w", err)
				continue
			}

			if req.Depth > c.cfg.maxDepth {
				if c.queue.Len() == 0 {
					signal.Cancel()
					c.queue.Cancel()
					continue
				}
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
						c.errors <- fmt.Errorf("store: %w", err)
						continue
					}
					c.errors <- fmt.Errorf("URL %s has been visited", target.URL.String())
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

			c.stream <- resp

			c.termLog.Infof("crawled: %s, depth: %d, status: %d, queue: %d", first64Chars(req.Target.URL.String()), req.Depth, resp.Status, c.queue.Len())

			if c.queue.Len() == 0 {
				signal.Cancel()
				c.queue.Cancel()
				continue
			}
		}
	}
}
func (c *Crawler) exit() {
	c.queue.Close()
	c.store.Close()
	c.fetcher.Close()
	c.termLog.Debugf("crawler closed")
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
