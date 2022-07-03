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
	queue   Queue
	store   Store
	log     Logger
	stream  chan Response
}

// NewWBot
func NewWBot(opts ...Option) *WBot {
	conf := &config{
		maxDepth:    10,
		parallel:    cores,
		maxBodySize: 1024 * 1024 * 10,
		userAgents:  newRotator([]string{}),
		proxies:     newRotator([]string{}),
	}

	wbot := &WBot{
		wg:      &sync.WaitGroup{},
		conf:    conf,
		fetcher: defaultFetcher(),
		limit:   newLimiter(1, 1),
		filter:  newFilter([]string{}, []string{}),
		store:   defaultStore[string](),
		queue:   defaultQueue[Request](),
		log:     nil,
		stream:  make(chan Response, cores),
	}

	// options
	wbot.SetOptions(opts...)

	return wbot
}

// Crawl
func (wb *WBot) Crawl(link string) error {
	// first request
	p := Param{
		Referer:     link,
		MaxBodySize: wb.conf.maxBodySize,
		UserAgent:   wb.conf.userAgents.next(),
		Proxy:       wb.conf.proxies.next(),
	}

	req, err := newRequest(link, 0, p)
	if err != nil {
		return err
	}

	// // no need to check first link
	// if wb.store.Visited(link) {
	// 	return fmt.Errorf("already visited")
	// }

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
		rep := newReport(resp, nil)
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
		// to avoid rotating user agent and proxy.
		p := Param{
			Referer:     req.URL.String(),
			MaxBodySize: wb.conf.maxBodySize,
		}
		nreq, err := newRequest(u.String(), 1, p)
		if err != nil {
			continue
		}

		if err := wb.queue.Enqueue(nreq); err != nil {
			continue
		}
	}

	// start crawl
	wb.wg.Add(wb.conf.parallel)
	for i := 0; i < wb.conf.parallel; i++ {
		go wb.crawl()
	}

	// wait for all workers to finish
	wb.wg.Wait()
	// wb.done()
	close(wb.stream)

	return nil
}

// crawl
func (wb *WBot) crawl() {
	defer wb.wg.Done()
	//
	for wb.queue.Next() {
		req, err := wb.queue.Dequeue()
		if err != nil {
			fmt.Println(err)
			time.Sleep(3 * time.Second)
			continue
		}

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

		req.Param.UserAgent = wb.conf.userAgents.next()
		req.Param.Proxy = wb.conf.proxies.next()

		// visit next url
		resp, err := wb.fetcher.Fetch(req)
		if err != nil {
			if wb.log != nil {
				rep := newReport(resp, err)
				wb.log.Send(rep)
			}
			continue
		}

		if wb.log != nil {
			rep := newReport(resp, nil)
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

			p := Param{
				Referer:     req.URL.String(),
				MaxBodySize: wb.conf.maxBodySize,
			}
			nreq, err := newRequest(u.String(), depth, p)
			if err != nil {
				continue
			}

			if err := wb.queue.Enqueue(nreq); err != nil {
				continue
			}
		}
	}
}

// SetOptions
func (wb *WBot) SetOptions(opts ...Option) {
	for _, opt := range opts {
		opt(wb)
	}
}

// Stream
func (wb *WBot) Stream() <-chan Response {
	return wb.stream
}

// Close
func (wb *WBot) Close() {
	wb.queue.Close()
	wb.store.Close()
	if wb.log != nil {
		wb.log.Close()
	}
}
