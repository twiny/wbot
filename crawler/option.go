package crawler

import (
	"github.com/twiny/poxa"
	"github.com/twiny/wbot"
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
func WithRateLimit(rates ...*wbot.RateLimit) Option {
	return func(c *Crawler) {
		c.limiter = newRateLimiter(rates...)
	}
}
func WithFilter(rules ...*wbot.FilterRule) Option {
	return func(c *Crawler) {
		c.filter = newFilter(rules...)
	}
}
func WithFetcher(fetcher wbot.Fetcher) Option {
	return func(c *Crawler) {
		c.fetcher = fetcher
	}
}
func WithQueue(queue wbot.Queue) Option {
	return func(c *Crawler) {
		c.queue = queue
	}
}
func WithStore(store wbot.Store) Option {
	return func(c *Crawler) {
		c.store = store
	}
}
func WithLogger(logger wbot.Logger) Option {
	return func(c *Crawler) {
		c.logger = logger
	}
}
