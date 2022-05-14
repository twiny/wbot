package wbot

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// default cpu core
var cores = func() int {
	c := runtime.NumCPU()
	if c == 1 {
		return c
	}
	return c - 1
}()

// WBot
type WBot struct {
	wg      *sync.WaitGroup
	conf    *config
	limit   *limiter
	filter  *filter
	fetcher Fetcher
	store   Store
	queue   Queue
	log     Logger
	stream  chan *Response
}

// NewWBot
func NewWBot(opts ...Option) (*WBot, error) {
	// default config
	conf := &config{
		maxDepth:    10,
		parallel:    cores,
		maxBodySize: 1024 * 1024 * 10,
	}

	wbot := &WBot{
		wg:      &sync.WaitGroup{},
		conf:    conf,
		fetcher: defaultFetcher(),
		limit:   newLimiter(1, 1),
		filter:  newFilter([]string{}, []string{}),
		store:   defaultStore[string](),
		queue:   defaultQueue[*Request](),
		log:     nil,
		stream:  make(chan *Response, cores),
	}

	for _, opt := range opts {
		opt(wbot)
	}

	return wbot, nil
}

// Crawl
func (wb *WBot) Crawl(link string) error {
	// first request
	p := param{
		referer:     link,
		maxBodySize: wb.conf.maxBodySize,
		userAgent:   wb.conf.userAgents.next(),
		proxy:       wb.conf.proxies.next(),
	}

	time.Sleep(5 * time.Second)

	req, err := NewRequest(link, 0, p)
	if err != nil {
		return err
	}

	if wb.store.Visited(link) {
		return fmt.Errorf("already visited")
	}

	// check filter
	if !wb.filter.Allow(req.URL) {
		return fmt.Errorf("not allowed")
	}

	// rate limit
	wb.limit.take(req.URL)

	resp, err := wb.fetcher.Fetch(req)
	if err != nil {
		return err
	}

	if wb.log != nil {
		rep := NewReport(resp, nil)
		wb.log.Send(rep)
	}

	// stream 1st response
	wb.stream <- resp

	// add to queue
	for _, link := range resp.NextURLs {
		u, err := req.AbsURL(link)
		if err != nil {
			continue
		}

		// is allowed domain
		if !strings.Contains(u.Hostname(), req.BaseDomain) {
			continue
		}

		// add only referer & maxBodySize
		// rest of params will be added
		// right before fetch request
		// to avoid running user agent and proxy rotation
		p := param{
			referer:     req.URL.String(),
			maxBodySize: wb.conf.maxBodySize,
		}
		nreq, err := NewRequest(u.String(), 1, p)
		if err != nil {
			continue
		}

		wb.queue.Add(nreq)
	}

	// start crawl
	wb.wg.Add(wb.conf.parallel)
	for i := 0; i < wb.conf.parallel; i++ {
		go func() {
			wb.crawl()
		}()
	}

	// wait for all workers to finish
	wb.wg.Wait()
	close(wb.stream)

	return nil
}

// crawl
func (wb *WBot) crawl() {
	defer func() {
		wb.wg.Done()
	}()

	//
	for wb.queue.Next() {
		req := wb.queue.Pop()

		// check if max depth reached
		if req.Depth > wb.conf.maxDepth {
			return
		}

		// if already visited
		if wb.store.Visited(req.URL.String()) {
			continue
		}

		// check filter
		if !wb.filter.Allow(req.URL) {
			continue
		}

		// rate limit
		wb.limit.take(req.URL)

		req.param.userAgent = wb.conf.userAgents.next()
		req.param.proxy = wb.conf.proxies.next()

		// visit next url
		resp, err := wb.fetcher.Fetch(req)
		if err != nil {
			if wb.log != nil {
				rep := NewReport(resp, err)
				wb.log.Send(rep)
			}
			continue
		}

		if wb.log != nil {
			rep := NewReport(resp, nil)
			wb.log.Send(rep)
		}

		// stream
		wb.stream <- resp

		// current depth
		depth := req.Depth
		// increment depth
		atomic.AddInt32(&depth, 1)

		// visit next urls
		for _, link := range resp.NextURLs {
			u, err := req.AbsURL(link)
			if err != nil {
				continue
			}

			// is allowed domain
			if !strings.Contains(u.Hostname(), req.BaseDomain) {
				continue
			}
			p := param{
				referer:     req.URL.String(),
				maxBodySize: wb.conf.maxBodySize,
			}
			nreq, err := NewRequest(u.String(), depth, p)
			if err != nil {
				continue
			}

			wb.queue.Add(nreq)
		}
	}
}

// Stream
func (wb *WBot) Stream() <-chan *Response {
	return wb.stream
}

// Close
func (wb *WBot) Close() {
	wb.queue.Close()
	wb.store.Close()
}
