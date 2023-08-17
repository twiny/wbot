package crawler

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/twiny/wbot"
	"github.com/twiny/wbot/plugin/fetcher"
	"github.com/twiny/wbot/plugin/queue"
	"github.com/twiny/wbot/plugin/store"

	clog "github.com/charmbracelet/log"
)

type (
	Crawler struct {
		wg sync.WaitGroup

		cfg *config

		fetcher wbot.Fetcher
		queue   wbot.Queue
		store   wbot.Store
		logger  wbot.Logger
		metrics wbot.MetricsMonitor

		filter  *filter
		limiter *rateLimiter
		robot   *robortManager

		stream chan *wbot.Response

		termLog *clog.Logger

		ctx    context.Context
		cancel context.CancelFunc
	}
)

func New(opts ...Option) *Crawler {
	ctx, cancel := context.WithCancel(context.Background())

	options := clog.Options{
		TimeFormat:      "2006-01-02 15:04:05",
		Level:           clog.DebugLevel,
		Prefix:          "[WBot]",
		ReportTimestamp: true,
	}

	c := &Crawler{
		cfg:     newConfig(-1, nil, nil, nil),
		filter:  newFilter(),
		limiter: newRateLimiter(),
		fetcher: fetcher.NewHTTPClient(),
		queue:   queue.NewInMemoryQueue(),
		store:   store.NewInMemoryStore(),
		// logger:  newFileLogger(),
		stream:  make(chan *wbot.Response, 2048),
		termLog: clog.NewWithOptions(os.Stdout, options),
		ctx:     ctx,
		cancel:  cancel,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.wg.Add(c.cfg.parallel)
	for i := 0; i < c.cfg.parallel; i++ {
		go c.routine()
	}

	return c
}

func (c *Crawler) SetOption(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
func (c *Crawler) Crawl(links ...string) {
	for _, link := range links {
		c.start(link)
	}
}
func (c *Crawler) Stream() <-chan *wbot.Response {
	return c.stream
}
func (c *Crawler) Wait() {
	c.wg.Wait()
}
func (c *Crawler) Done() {
	c.cancel()
}

func (c *Crawler) start(raw string) {
	target, err := wbot.ValidURL(raw)
	if err != nil {
		c.termLog.Errorf("start: %s", err.Error())
		return
	}

	// first request
	param := &wbot.Param{
		MaxBodySize: c.cfg.maxBodySize,
		UserAgent:   c.cfg.userAgents.Next(),
	}

	if c.cfg.proxies != nil {
		param.Proxy = c.cfg.proxies.Next()
	}

	key, err := wbot.HashLink(target.String())
	if err != nil {
		// todo: log
		// review: why error at this point? link is already validated
		c.termLog.Errorf("hashlink -> invalid url: %s\n", target)
		return
	}

	hostname, err := wbot.Hostname(target.String())
	if err != nil {
		// todo: log
		c.termLog.Errorf("hostname -> invalid url: %s\n", target)
		return
	}

	req := &wbot.Request{
		ID:       key,
		BaseHost: hostname,
		URL:      target,
		Depth:    0,
		Param:    param,
	}

	c.termLog.Infof("start %+v\n", req)

	// todo: fix robots.txt
	// if !c.robot.Allowed(ua, target) {
	// 	// todo: log
	// 	return
	// }

	c.limiter.wait(target) // should be unique hash

	resp, err := c.fetcher.Fetch(context.TODO(), req)
	if err != nil {
		// todo: log
		c.termLog.Errorf("fetcher -> %s\n", err.Error())
		return
	}

	_, _ = c.store.HasVisited(context.TODO(), key)

	c.stream <- resp // stream 1st response

	for _, link := range resp.NextURLs {
		u, err := req.ResolveURL(link)
		if err != nil {
			c.termLog.Errorf("resolve url -> %s\n", err.Error())
			continue
		}

		// todo: this should only allow base host
		if !strings.Contains(u.Hostname(), req.BaseHost) {
			c.termLog.Errorf("invalid hostname: %s\n", u.Hostname())
			continue
		}

		// add only referer & maxBodySize
		// rest of params will be added
		// right before fetch request
		// to avoid rotating user agent and proxy.
		nextParm := &wbot.Param{
			Referer:     req.URL.String(),
			MaxBodySize: c.cfg.maxBodySize,
		}

		nextReq := &wbot.Request{
			ID:       key,
			BaseHost: hostname,
			URL:      u,
			Depth:    req.Depth + 1,
			Param:    nextParm,
		}

		if err := c.queue.Push(context.TODO(), nextReq); err != nil {
			c.termLog.Errorf("push -> %s\n", err.Error())
			continue
		}
	}
}
func (c *Crawler) routine() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			req, err := c.queue.Pop(context.TODO())
			if err != nil {
				if err == queue.ErrQueueClosed {
					c.termLog.Errorf("queue closed\n")
					return
				}
				c.termLog.Errorf("pop -> %s\n", err.Error())
				continue
			}

			if visited, err := c.store.HasVisited(context.TODO(), req.ID); visited {
				if err != nil {
					// todo: log
					c.termLog.Errorf("has visited -> %s\n", err.Error())
					continue
				}
				// todo: log
				c.termLog.Errorf("already visited: %s\n", req.URL)
				continue
			}

			if req.Depth > c.cfg.maxDepth {
				// todo: log
				c.termLog.Errorf("max depth reached: %s\n", req.URL)
				continue
			}

			// if !c.robot.Allowed(req.Param.UserAgent, req.URL.String()) {
			// 	// todo: log
			// 	continue
			// }

			if !c.filter.allow(req.URL) {
				// todo: log
				c.termLog.Errorf("filter -> %s\n", req.URL)
				continue
			}

			c.limiter.wait(req.URL)

			resp, err := c.fetcher.Fetch(c.ctx, req)
			if err != nil {
				// todo: log
				c.termLog.Errorf("fetcher -> %s\n", err.Error())
				continue
			}

			for _, link := range resp.NextURLs {
				u, err := req.ResolveURL(link)
				if err != nil {
					c.termLog.Errorf("resolve url -> %s\n", err.Error())
					continue
				}

				key, err := wbot.HashLink(u.String())
				if err != nil {
					// todo: log
					c.termLog.Errorf("hashlink -> %s\n", err.Error())
					continue
				}
				hostname, err := wbot.Hostname(u.String())
				if err != nil {
					// todo: log
					c.termLog.Errorf("hostname -> %s\n", err.Error())
					continue
				}

				nextReq := &wbot.Request{
					ID:       key,
					BaseHost: hostname,
					URL:      u,
					Depth:    req.Depth + 1,
					Param:    req.Param,
				}

				if err := c.queue.Push(context.TODO(), nextReq); err != nil {
					// todo: log
					c.termLog.Errorf("push -> %s\n", err.Error())
					continue
				}
			}

			// if c.log != nil {
			// 	rep := newReport(resp, nil)
			// 	c.log.Send(rep)
			// }

			// stream
			c.stream <- resp

			c.termLog.Errorf("crawled: %s\n", req.URL)
		}
	}
}
