package crawler

import (
	"context"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/twiny/wbot"
	"github.com/twiny/wbot/plugin/fetcher"
	"github.com/twiny/wbot/plugin/store"

	"github.com/twiny/flare"

	clog "github.com/charmbracelet/log"
)

type (
	Crawler struct {
		wg sync.WaitGroup

		cfg *config

		fetcher wbot.Fetcher
		store   wbot.Store
		logger  wbot.Logger
		metrics wbot.MetricsMonitor

		filter  *filter
		limiter *rateLimiter
		robot   *robortManager

		counter int32
		queue   chan *wbot.Request
		stream  chan *wbot.Response

		finished flare.Notifier
		quit     flare.Notifier

		termLog *clog.Logger
	}

	Reader func(*wbot.Response) error
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
		// metrics: newMetricsMonitor(),

		filter:  newFilter(),
		limiter: newRateLimiter(),
		// robot:   newRobotsManager(),

		queue:  make(chan *wbot.Request, 1024),
		stream: make(chan *wbot.Response, 1024),

		termLog: clog.NewWithOptions(os.Stdout, logOpt),

		finished: flare.New(),
		quit:     flare.New(),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.wg.Add(c.cfg.parallel)
	c.termLog.Infof("starting %d workers...", c.cfg.parallel)
	for i := 0; i < c.cfg.parallel; i++ {
		go c.crawler()
	}

	return c
}

func (c *Crawler) SetOption(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
func (c *Crawler) Crawl(links ...string) {
	var targets []*url.URL
	for _, link := range links {
		target, err := wbot.ValidURL(link)
		if err != nil {
			c.termLog.Errorf("start: %s", err.Error())
			continue
		}
		targets = append(targets, target)
	}

	if len(targets) == 0 {
		c.termLog.Errorf("no valid links found")
		c.finished.Cancel()
		c.quit.Cancel()
		c.wg.Wait()
		close(c.queue)
		close(c.stream)
		return
	}

	c.termLog.Infof("crawling %d links...", len(targets))

	for _, target := range targets {
		c.start(target)
	}

	c.wg.Wait()
}
func (c *Crawler) Read(fn Reader) error {
	for {
		select {
		case <-c.quit.Done():
			return nil
		case <-c.finished.Done():
			return nil
		case resp := <-c.stream:
			if err := fn(resp); err != nil {
				return err
			}
		}
	}
}
func (c *Crawler) Close() {
	c.termLog.Infof("done: crawled %d links", c.counter)
	c.finished.Cancel()
	c.quit.Cancel()
	c.wg.Wait()
	c.store.Close()
	c.fetcher.Close()
}

func (c *Crawler) start(target *url.URL) {
	// first request
	param := &wbot.Param{
		MaxBodySize: c.cfg.maxBodySize,
		UserAgent:   c.cfg.userAgents.Next(),
	}

	if c.cfg.proxies != nil {
		param.Proxy = c.cfg.proxies.Next()
	}

	hostname, err := wbot.Hostname(target.Hostname())
	if err != nil {
		// todo: log
		c.termLog.Errorf("hostname -> invalid url: %s", target)
		return
	}

	req := &wbot.Request{
		BaseHost: hostname,
		URL:      target,
		Depth:    1,
		Param:    param,
	}

	atomic.AddInt32(&c.counter, 1)
	c.queue <- req
}
func (c *Crawler) crawler() {
	defer c.wg.Done()

	c.termLog.Debugf("worker started")
	for {
		select {
		case <-c.quit.Done():
			c.termLog.Debugf("worker quit")
			return
		case <-c.finished.Done():
			c.termLog.Debugf("worker finished")
			return
			// continue
		case req := <-c.queue:
			if req.Depth > c.cfg.maxDepth {
				atomic.AddInt32(&c.counter, -1)

				if c.counter == 0 {
					c.termLog.Infof("done: crawled %d links", c.counter)
					c.finished.Cancel()
					c.quit.Cancel()
					return
				}
				// c.termLog.Errorf("max depth reached: %s, counter: %d, queue: %d", first64Chars(req.URL.String()), c.counter, len(c.queue))
				continue
			}

			c.limiter.wait(req.URL)

			resp, err := c.fetcher.Fetch(context.TODO(), req)
			if err != nil {
				// todo: log
				atomic.AddInt32(&c.counter, -1)
				c.termLog.Errorf("fetcher -> %s", err.Error())
				continue
			}

			atomic.AddInt32(&req.Depth, 1)
			for _, link := range resp.NextURLs {
				u, err := req.ResolveURL(link)
				if err != nil {
					// c.termLog.Errorf("crwal: resolve url %s -> %s", link, err.Error())
					continue
				}

				hostname, err := wbot.Hostname(u.Hostname())
				if err != nil {
					// todo: log
					// c.termLog.Errorf("hostname -> %s", err.Error())
					continue
				}

				if !strings.Contains(u.Hostname(), hostname) {
					// todo: log
					// c.termLog.Errorf("invalid hostname: %s", u)
					continue
				}

				// if !c.robot.Allowed(req.Param.UserAgent, req.URL.String()) {
				// 	// todo: log
				// 	continue
				// }

				if !c.filter.allow(u) {
					// todo: log
					// c.termLog.Errorf("filter -> %s", req.URL)
					continue
				}

				if visited, err := c.store.HasVisited(context.TODO(), u.String()); visited {
					if err != nil {
						// todo: log
						// c.termLog.Errorf("has visited -> %s", err.Error())
						// continue
					}
					// todo: log
					// c.termLog.Errorf("already visited: %s", req.URL)
					continue
				}

				nextReq := &wbot.Request{
					BaseHost: hostname,
					URL:      u,
					Depth:    req.Depth,
					Param:    req.Param,
				}

				atomic.AddInt32(&c.counter, 1)
				c.queue <- nextReq
			}

			// if c.log != nil {
			// 	rep := newReport(resp, nil)
			// 	c.log.Send(rep)
			// }

			// stream
			c.stream <- resp
			atomic.AddInt32(&c.counter, -1)

			c.termLog.Infof("crawled: %s, depth: %d, counter: %d, queue: %d", first64Chars(req.URL.String()), req.Depth, c.counter, len(c.queue))
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
