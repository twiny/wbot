package wbot

import (
	"net/url"
	"time"

	"github.com/twiny/ratelimit"
)

// Limiter
type limiter struct {
	rate     int
	duration time.Duration
	list     map[string]*ratelimit.Limiter
}

// newLimiter
func newLimiter(r int, d time.Duration) *limiter {
	return &limiter{
		rate:     r,
		duration: d,
		list:     make(map[string]*ratelimit.Limiter),
	}
}

// Take
func (l *limiter) take(u *url.URL) {
	hostname := u.Hostname()

	limit, found := l.list[hostname]
	if !found {
		limit = ratelimit.NewLimiter(l.rate, l.duration)
		l.list[hostname] = limit
	}

	limit.Take()
}
