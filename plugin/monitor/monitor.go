package monitor

import (
	"sync/atomic"
)

type (
	metricsMonitor struct {
		totalRequests      int64
		successfulRequests int64
		failedRequests     int64
		retries            int64
		redirects          int64
		totalPages         int64
		crawledPages       int64
		skippedPages       int64
		parsedLinks        int64
		clientErrors       int64
		serverErrors       int64
	}
)

func NewmetricsMonitor() *metricsMonitor {
	return &metricsMonitor{}
}

func (m *metricsMonitor) IncrementTotalRequests() {
	atomic.AddInt64(&m.totalRequests, 1)
}
func (m *metricsMonitor) IncrementSuccessfulRequests() {
	atomic.AddInt64(&m.successfulRequests, 1)
}
func (m *metricsMonitor) IncrementFailedRequests() {
	atomic.AddInt64(&m.failedRequests, 1)
}
func (m *metricsMonitor) IncrementRetries() {
	atomic.AddInt64(&m.retries, 1)
}
func (m *metricsMonitor) IncrementRedirects() {
	atomic.AddInt64(&m.redirects, 1)
}
func (m *metricsMonitor) IncrementTotalPages() {
	atomic.AddInt64(&m.totalPages, 1)
}
func (m *metricsMonitor) IncrementCrawledPages() {
	atomic.AddInt64(&m.crawledPages, 1)
}
func (m *metricsMonitor) IncrementSkippedPages() {
	atomic.AddInt64(&m.skippedPages, 1)
}
func (m *metricsMonitor) IncrementParsedLinks() {
	atomic.AddInt64(&m.parsedLinks, 1)
}
func (m *metricsMonitor) IncrementClientErrors() {
	atomic.AddInt64(&m.clientErrors, 1)
}
func (m *metricsMonitor) IncrementServerErrors() {
	atomic.AddInt64(&m.serverErrors, 1)
}
