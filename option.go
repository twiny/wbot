package wbot

import (
	"github.com/rs/zerolog"
	"github.com/twiny/poxa"

	"github.com/twiny/wbot/pkg/api"
)

type (
	Option func(c *Crawler)
)

func WithParallel(parallel int) Option {
	return func(c *Crawler) {
		c.cfg.parallel = parallel
	}
}
func WithMaxDepth(maxDepth int32) Option {
	return func(c *Crawler) {
		c.cfg.maxDepth = maxDepth
	}
}
func WithUserAgents(userAgents []string) Option {
	return func(c *Crawler) {
		c.cfg.userAgents = poxa.NewSpinner(userAgents...)
	}
}
func WithProxies(proxies []string) Option {
	return func(c *Crawler) {
		c.cfg.proxies = poxa.NewSpinner(proxies...)
	}
}
func WithRateLimit(rates ...*api.RateLimit) Option {
	return func(c *Crawler) {
		c.limiter = newRateLimiter(rates...)
	}
}
func WithFilter(rules ...*api.FilterRule) Option {
	return func(c *Crawler) {
		c.filter = newFilter(rules...)
	}
}
func WithFetcher(fetcher api.Fetcher) Option {
	return func(c *Crawler) {
		c.fetcher = fetcher
	}
}
func WithStore(store api.Store) Option {
	return func(c *Crawler) {
		c.store = store
	}
}
func WithQueue(queue api.Queue) Option {
	return func(c *Crawler) {
		c.queue = queue
	}
}
func WithLogLevel(level zerolog.Level) Option {
	return func(c *Crawler) {
		c.logger = c.logger.Level(level)
	}
}
