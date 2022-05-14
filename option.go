package wbot

import (
	"time"
)

// Option
type Option func(*WBot) error

// SetFetcher
func SetFetcher(f Fetcher) Option {
	return func(w *WBot) error {
		w.fetcher = f
		return nil
	}
}

// SetStore
func SetStore(s Store) Option {
	return func(w *WBot) error {
		w.store = s
		return nil
	}
}

// SetQueue
func SetQueue(q Queue) Option {
	return func(w *WBot) error {
		w.queue = q
		return nil
	}
}

// SetLogger
func SetLogger(l Logger) Option {
	return func(w *WBot) error {
		w.log = l
		return nil
	}
}

// SetLimiter
func SetRateLimit(rate int, interval time.Duration) Option {
	return func(w *WBot) error {
		w.limit = newLimiter(rate, interval)
		return nil
	}
}

// SetFilter
func SetFilter(allowed, disallowed []string) Option {
	return func(w *WBot) error {
		w.filter = newFilter(allowed, disallowed)
		return nil
	}
}

// SetMaxDepth
func SetMaxDepth(depth int32) Option {
	return func(w *WBot) error {
		w.conf.maxDepth = depth
		return nil
	}
}

// SetParallel
func SetParallel(parallel int) Option {
	return func(w *WBot) error {
		w.conf.parallel = parallel
		return nil
	}
}

// SetMaxBodySize
func SetMaxBodySize(size int64) Option {
	return func(w *WBot) error {
		w.conf.maxBodySize = size
		return nil
	}
}

// SetUserAgents
func SetUserAgents(agents []string) Option {
	return func(w *WBot) error {
		w.conf.userAgents = newRotator(agents)
		return nil
	}
}

// SetProxies
func SetProxies(proxies []string) Option {
	return func(w *WBot) error {
		w.conf.proxies = newRotator(proxies)
		return nil
	}
}
