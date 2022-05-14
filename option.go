package wbot

import (
	"time"
)

// Option
type Option func(*WBot)

// SetFetcher
func SetFetcher(f Fetcher) Option {
	return func(w *WBot) {
		w.fetcher = f
	}
}

// SetStore
func SetStore(s Store) Option {
	return func(w *WBot) {
		w.store = s
	}
}

// SetQueue
func SetQueue(q Queue) Option {
	return func(w *WBot) {
		w.queue = q
	}
}

// SetLogger
func SetLogger(l Logger) Option {
	return func(w *WBot) {
		w.log = l
	}
}

// SetLimiter
func SetRateLimit(rate int, interval time.Duration) Option {
	return func(w *WBot) {
		w.limit = newLimiter(rate, interval)
	}
}

// SetFilter
func SetFilter(allowed, disallowed []string) Option {
	return func(w *WBot) {
		w.filter = newFilter(allowed, disallowed)
	}
}

// SetMaxDepth
func SetMaxDepth(depth int32) Option {
	return func(w *WBot) {
		w.conf.maxDepth = depth
	}
}

// SetParallel
func SetParallel(parallel int) Option {
	return func(w *WBot) {
		w.conf.parallel = parallel
	}
}

// SetMaxBodySize
func SetMaxBodySize(size int64) Option {
	return func(w *WBot) {
		w.conf.maxBodySize = size
	}
}

// SetUserAgents
func SetUserAgents(agents []string) Option {
	return func(w *WBot) {
		w.conf.userAgents = newRotator(agents)
	}
}

// SetProxies
func SetProxies(proxies []string) Option {
	return func(w *WBot) {
		w.conf.proxies = newRotator(proxies)
	}
}
