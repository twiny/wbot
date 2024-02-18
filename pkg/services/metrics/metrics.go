package metrics

import (
	"sync/atomic"
)

type (
	metricsMonitor struct {
		totalRequests      int64
		successfulRequests int64
		failedRequests     int64

		totalLink      int64
		crawledLink    int64
		skippedLink    int64
		duplicatedLink int64
	}
)

func NewMetricsMonitor() *metricsMonitor {
	return &metricsMonitor{}
}

func (m *metricsMonitor) IncTotalRequests() {
	atomic.AddInt64(&m.totalRequests, 1)
}
func (m *metricsMonitor) IncSuccessfulRequests() {
	atomic.AddInt64(&m.successfulRequests, 1)
}
func (m *metricsMonitor) IncFailedRequests() {
	atomic.AddInt64(&m.failedRequests, 1)
}
func (m *metricsMonitor) IncTotalLink() {
	atomic.AddInt64(&m.totalLink, 1)
}
func (m *metricsMonitor) IncCrawledLink() {
	atomic.AddInt64(&m.crawledLink, 1)
}
func (m *metricsMonitor) IncSkippedLink() {
	atomic.AddInt64(&m.skippedLink, 1)
}
func (m *metricsMonitor) IncDuplicatedLink() {
	atomic.AddInt64(&m.duplicatedLink, 1)
}
func (m *metricsMonitor) Metrics() map[string]int64 {
	return map[string]int64{
		"total_requests":      atomic.LoadInt64(&m.totalRequests),
		"successful_requests": atomic.LoadInt64(&m.successfulRequests),
		"failed_requests":     atomic.LoadInt64(&m.failedRequests),
		"total_link":          atomic.LoadInt64(&m.totalLink),
		"crawled_link":        atomic.LoadInt64(&m.crawledLink),
		"skipped_link":        atomic.LoadInt64(&m.skippedLink),
		"duplicated_link":     atomic.LoadInt64(&m.duplicatedLink),
	}
}
