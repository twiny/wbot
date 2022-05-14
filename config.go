package wbot

// Config
type config struct {
	maxDepth    int32
	parallel    int
	maxBodySize int64
	userAgents  *rotator
	proxies     *rotator
}
