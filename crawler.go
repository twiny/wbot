package wbot

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
	"github.com/twiny/wbot/pkg/api"
	"github.com/twiny/wbot/pkg/services/fetcher"
	"github.com/twiny/wbot/pkg/services/metrics"
	"github.com/twiny/wbot/pkg/services/queue"
	"github.com/twiny/wbot/pkg/services/store"
)

const (
	crawlRunning = 0
	crawlStopped = 1
)

type (
	Crawler struct {
		wg  *sync.WaitGroup
		cfg *config

		fetcher api.Fetcher
		store   api.Store
		queue   api.Queue
		metrics api.MetricsMonitor

		filter  *filter
		limiter *rateLimiter
		robot   *robotManager

		stream chan *api.Response

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
		metrics: metrics.NewMetricsMonitor(),

		filter:  newFilter(),
		limiter: newRateLimiter(),
		robot:   newRobotManager(false),

		stream: make(chan *api.Response, 1024),

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

func (c *Crawler) Run(links ...string) error {
	var (
		targets []*api.ParsedURL
		errs    []error
	)

	for _, link := range links {
		target, err := api.NewURL(link)
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
		c.add(target)
	}

	c.status = crawlRunning

	c.logger.Info().Msgf("Starting crawler with %d links", len(targets))

	c.wg.Add(c.cfg.parallel)
	for i := 0; i < c.cfg.parallel; i++ {
		go c.crawl(i)
	}

	c.wg.Wait()
	return nil
}
func (c *Crawler) OnReponse(fn func(*api.Response)) {
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
func (c *Crawler) Metrics() map[string]int64 {
	return c.metrics.Metrics()
}
func (c *Crawler) Shutdown() {
	c.stop()
}

func (c *Crawler) add(target *api.ParsedURL) {
	param := &api.Param{
		MaxBodySize: c.cfg.maxBodySize,
		UserAgent:   c.cfg.userAgents.Next(),
		Timeout:     c.cfg.timeout,
	}

	if c.cfg.proxies != nil {
		param.Proxy = c.cfg.proxies.Next()
	}

	req := &api.Request{
		Target: target,
		Param:  param,
		Depth:  0,
	}

	if err := c.queue.Push(c.ctx, req); err != nil {
		c.logger.Err(err).Msgf("pop")
		return
	}
}
func (c *Crawler) crawl(id int) {
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

			// not elegant, but just to give the queue some time to fill up
			if atomic.LoadInt32(&c.status) == crawlRunning && c.queue.Len() == 0 {
				<-time.After(1 * time.Second)
				continue
			}

			req, err := c.queue.Pop(c.ctx)
			if err != nil {
				c.logger.Err(err).Msgf("pop")
				continue
			}
			c.metrics.IncTotalRequests()

			// if the next response will exceed the max depth,
			// we signal the crawler to stop
			if atomic.LoadInt32(&req.Depth) > c.cfg.maxDepth-1 {
				atomic.StoreInt32(&c.status, crawlStopped)
			}

			c.limiter.wait(req.Target)

			resp, err := c.fetcher.Fetch(c.ctx, req)
			if err != nil {
				c.metrics.IncFailedRequests()
				c.logger.Err(err).Any("target", req.Target.String()).Msgf("fetch")
				continue
			}

			c.stream <- resp
			c.metrics.IncSuccessfulRequests()

			c.logger.Debug().Any("target", req.Target.String()).Msgf("fetched")

			// increment the depth for the next requests
			nextDepth := atomic.AddInt32(&req.Depth, 1)

			if nextDepth > c.cfg.maxDepth {
				continue
			}

			// logging here will just flood the logs
			for _, target := range resp.NextURLs {
				c.metrics.IncTotalLink()

				if !strings.Contains(target.URL.Host, req.Target.Root) {
					c.metrics.IncSkippedLink()
					continue
				}

				if !c.robot.Allowed(req.Param.UserAgent, req.Target.URL.String()) {
					c.metrics.IncSkippedLink()
					// todo: log
					continue
				}

				if !c.filter.allow(target) {
					c.metrics.IncSkippedLink()
					continue
				}

				if visited, err := c.store.HasVisited(c.ctx, target); visited {
					if err != nil {
						c.logger.Err(err).Msgf("store")
					}
					c.metrics.IncDuplicatedLink()
					continue
				}

				nextReq := &api.Request{
					Target: target,
					Depth:  req.Depth,
					Param:  req.Param,
				}

				if err := c.queue.Push(c.ctx, nextReq); err != nil {
					c.logger.Err(err).Any("target", target.String()).Msgf("push")
					continue
				}

				c.metrics.IncCrawledLink()
			}
		}
	}
}
