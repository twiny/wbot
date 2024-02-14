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

func (m *metricsMonitor) IncTotalRequests() {
	atomic.AddInt64(&m.totalRequests, 1)
}
func (m *metricsMonitor) IncSuccessfulRequests() {
	atomic.AddInt64(&m.successfulRequests, 1)
}
func (m *metricsMonitor) IncFailedRequests() {
	atomic.AddInt64(&m.failedRequests, 1)
}
func (m *metricsMonitor) IncRetries() {
	atomic.AddInt64(&m.retries, 1)
}
func (m *metricsMonitor) IncRedirects() {
	atomic.AddInt64(&m.redirects, 1)
}
func (m *metricsMonitor) IncTotalPages() {
	atomic.AddInt64(&m.totalPages, 1)
}
func (m *metricsMonitor) IncCrawledPages() {
	atomic.AddInt64(&m.crawledPages, 1)
}
func (m *metricsMonitor) IncSkippedPages() {
	atomic.AddInt64(&m.skippedPages, 1)
}
func (m *metricsMonitor) IncParsedLinks() {
	atomic.AddInt64(&m.parsedLinks, 1)
}
func (m *metricsMonitor) IncClientErrors() {
	atomic.AddInt64(&m.clientErrors, 1)
}
func (m *metricsMonitor) IncServerErrors() {
	atomic.AddInt64(&m.serverErrors, 1)
}
