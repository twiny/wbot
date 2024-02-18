package crawler

import (
	"strconv"
	"strings"
	"time"

	"github.com/twiny/ratelimit"
	"github.com/twiny/wbot"
)

var (
	defaultRateLimit = "10/1s"
)

type (
	rateLimiter struct {
		table map[string]*ratelimit.Limiter
	}
)

func newRateLimiter(limits ...*wbot.RateLimit) *rateLimiter {
	rl := &rateLimiter{
		table: make(map[string]*ratelimit.Limiter),
	}

	// Handle the default wildcard limit.
	hasWildcard := false
	if len(limits) > 0 {
		for _, limit := range limits {
			if limit.Hostname == "*" {
				hasWildcard = true
				break
			}
		}
	}

	if !hasWildcard {
		limits = append(limits, &wbot.RateLimit{
			Hostname: "*",
			Rate:     defaultRateLimit,
		})
	}

	for _, rate := range limits {
		r, l := parseRateLimit(rate.Rate)
		rl.table[rate.Hostname] = ratelimit.NewLimiter(r, l)
	}

	return rl
}
func (l *rateLimiter) wait(u *wbot.ParsedURL) {
	limit, found := l.table[u.Root]
	if !found {
		limit = l.table["*"]
	}

	limit.Take()
}

func parseRateLimit(s string) (rate int, interval time.Duration) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return parseRateLimit(defaultRateLimit)
	}

	rate, err := strconv.Atoi(parts[0])
	if err != nil {
		return parseRateLimit(defaultRateLimit)
	}

	intervalValueStr := parts[1][:len(parts[1])-1]
	intervalValue, err := strconv.Atoi(intervalValueStr)
	if err != nil {
		return parseRateLimit(defaultRateLimit)
	}

	switch parts[1][len(parts[1])-1] {
	case 's', 'S':
		interval = time.Duration(intervalValue) * time.Second
	case 'm', 'M':
		interval = time.Duration(intervalValue) * time.Minute
	case 'h', 'H':
		interval = time.Duration(intervalValue) * time.Hour
	default:
		return parseRateLimit(defaultRateLimit)
	}

	return rate, interval
}
