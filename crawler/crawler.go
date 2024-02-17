package crawler

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
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

		status int32
		flare  flare.Notifier

		logger zerolog.Logger

		ctx  context.Context
		stop context.CancelFunc
	}
)

func New(opts ...Option) *Crawler {
	cw := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	logger := zerolog.New(cw).With().Timestamp().Logger()

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

		status: crawlStopped,
		flare:  flare.New(),
		logger: logger,

		ctx:  ctx,
		stop: stop,
	}

	for _, opt := range opts {
		opt(c)
	}

	// this routine waits for quit signal
	go func() {
		<-c.ctx.Done()
		c.logger.Info().Msgf("Crawler is shutting down")

		c.flare.Cancel()
		c.queue.Close()
		c.store.Close()
		c.fetcher.Close()

		c.wg.Wait()
		close(c.stream)
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

	c.logger.Info().Msgf("Starting crawler with %d links", len(targets))

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
		c.logger.Err(err).Msgf("pop")
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

			// not elegant, but if the queue is empty, we wait for a while
			if atomic.LoadInt32(&c.status) == crawlRunning && c.queue.Len() == 0 {
				<-time.After(1 * time.Second)
				continue
			}

			req, err := c.queue.Pop(c.ctx)
			if err != nil {
				c.logger.Err(err).Msgf("pop")
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
				c.logger.Err(err).Any("target", req.Target.String()).Msgf("fetch")
				continue
			}

			c.logger.Debug().Msgf("Fetched: %s", resp.URL.String())

			c.stream <- resp

			atomic.AddInt32(&req.Depth, 1)

			if atomic.LoadInt32(&req.Depth) > c.cfg.maxDepth {
				continue
			}

			// logging here will just flood the logs
			for _, target := range resp.NextURLs {
				if !strings.Contains(target.URL.Host, req.Target.Root) {
					continue
				}

				// if !c.robot.Allowed(req.Param.UserAgent, req.URL.String()) {
				// 	// todo: log
				// 	continue
				// }

				if !c.filter.allow(target) {
					continue
				}

				if visited, err := c.store.HasVisited(c.ctx, target); visited {
					if err != nil {
						c.logger.Err(err).Msgf("store")
					}
					continue
				}

				nextReq := &wbot.Request{
					Target: target,
					Depth:  req.Depth,
					Param:  req.Param,
				}

				if err := c.queue.Push(c.ctx, nextReq); err != nil {
					c.logger.Err(err).Any("target", target.String()).Msgf("push")
					continue
				}
			}
		}
	}
}
