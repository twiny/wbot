## WBot - a web crawler

A configurable, thread-safe web crawler, provides a minimal interface for crawling and downloading web pages.

### Features:
- Clean minimal API.
- Configurable: MaxDepth, MaxBodySize, Rate Limit, Parrallelism,  User Agent & Proxy rotation.
- Memory-efficient, thread-safe.
- Provides built-in interface: Fetcher, Store, Queue & a Logger.

### WBot Specifications:

#### Interfaces
```go
// Fetcher
type Fetcher interface {
	Fetch(req *Request) (*Response, error)
	Close() error
}

// Store
type Store interface {
	Visited(link string) bool
	Close() error
}

// Queue
type Queue interface {
	Add(req *Request)
	Pop() *Request
	Next() bool
	Close() error
}

// Logger
type Logger interface {
	Send(rep *Report)
	Close() error
}
```

#### API
```go
// NewWBot
func NewWBot(opts ...Option) (*WBot, error)

// Crawl
func (wb *WBot) Crawl(link string) error

// SetOptions
func (wb *WBot) SetOptions(opts ...Option)

// Stream
func (wb *WBot) Stream() <-chan *Response

// Close
func (wb *WBot) Close() 
```

### Installation
requires Go1.18

`go get github.com/twiny/wbot`

### Example
```go
package main

import (
	"fmt"
	"time"

	"github.com/twiny/wbot"
)

//
func main() {
	// options
	opts := []wbot.Option{
		wbot.SetMaxDepth(5),
		wbot.SetParallel(10),
		wbot.SetRateLimit(1, 1*time.Second),
		wbot.SetMaxBodySize(1024 * 1024),
		wbot.SetUserAgents([]string{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"}),
	}

	// wbot
	bot := wbot.NewWBot(opts...)
	defer bot.Close()

	// crawl
	site := `https://www.github.com`

	// stream
	// stream
	go func() {
		count := 0
		for resp := range bot.Stream() {
			count++
			fmt.Printf("num: %d - depth: %d - visited url:%s - status:%d - body len: %d\n", count, resp.Depth, resp.URL.String(), resp.Status, len(resp.Body))
		}
	}()

	if err := bot.Crawl(site); err != nil {
		panic(err)
	}

	fmt.Println("done")
}
```

### TODO
- [ ] Add support for robots.txt.
- [ ] Add test cases.
- [ ] Implement `Fetch` using Chromedp.
- [ ] Add more examples.
- [ ] Add documentation.

### Bugs
Bugs or suggestions? Please visit the [issue tracker](https://github.com/twiny/wbot/issues).
