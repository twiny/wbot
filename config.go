package wbot

// Config
type config struct {
	maxDepth    int32
	parallel    int
	maxBodySize int64
	userAgents  *rotator
	proxies     *rotator
}

// NewConfig
func NewConfig() *config {
	return &config{
		maxDepth:    10,
		parallel:    cores,
		maxBodySize: 1024 * 1024 * 10,
		userAgents:  newRotator([]string{}),
		proxies:     newRotator([]string{}),
	}
}
