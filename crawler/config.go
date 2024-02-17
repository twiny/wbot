package crawler

import (
	"runtime"
	"time"

	"github.com/twiny/poxa"
)

const (
	defaultReferrer    = "https://www.google.com/search"
	defaultUserAgent   = "WBot/0.1.6 (+https://github.com/twiny/wbot)"
	defaultTimeout     = 10 * time.Second
	defaultMaxBodySize = int64(1024 * 1024 * 5) // 5MB
)

type (
	config struct {
		parallel    int
		maxDepth    int32
		maxBodySize int64
		timeout     time.Duration
		userAgents  poxa.Spinner[string]
		referrers   poxa.Spinner[string]
		proxies     poxa.Spinner[string]
	}
)

func newConfig(maxDepth int32, userAgents, referrers, proxies []string) *config {
	if maxDepth <= 0 {
		maxDepth = 10
	}

	var conf = &config{
		parallel:    runtime.NumCPU(),
		maxDepth:    maxDepth,
		maxBodySize: defaultMaxBodySize,
		timeout:     defaultTimeout,
		userAgents:  poxa.NewSpinner(defaultUserAgent),
		referrers:   poxa.NewSpinner(defaultReferrer),
		proxies:     nil,
	}

	if len(userAgents) > 0 {
		uaList := poxa.NewSpinner(userAgents...)
		if uaList != nil {
			conf.userAgents = uaList
		}
	}

	if len(referrers) > 0 {
		refList := poxa.NewSpinner(referrers...)
		if refList != nil {
			conf.referrers = refList
		}
	}

	if len(proxies) > 0 {
		proxyList := poxa.NewSpinner(proxies...)
		if proxyList != nil {
			conf.proxies = proxyList
		}
	}

	return conf
}
