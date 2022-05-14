## WBot - a web crawler

A configurable, thread-safe web crawler, provides a minimal interface for crawling and downloading web pages.

### Features:
- clean minimal API.
- Configurable: MaxDepth, MaxBodySize, Rate Limit, Parrallelism,  User Agent & Proxy rotation.
- Memory-efficient, thread-safe.
- Provides built-in interface: Fetcher, Store, Queue & a Logger.

### WBot Specifications:

#### Interfaces
```go
// Fetcher
type Fetcher interface {
	Fetch(req *Request) (*Response, error)
}

// Store
type Store interface {
	Visited(link string) bool
	Close()
}

// Queue
type Queue interface {
	Add(req *Request)
	Pop() *Request
	Next() bool
	Close()
}

// Logger
type Logger interface {
	Send(rep *Report)
}
```

#### API
```go
// NewWBot
func NewWBot(opts ...Option) (*WBot, error)

// Crawl
func (wb *WBot) Crawl(link string) error

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
	"log"
	"time"

	"github.com/twiny/wbot"
)

func main() {
	// options
	opts := []wbot.Option{
		wbot.SetMaxDepth(5),
		wbot.SetRateLimit(1, 2*time.Second),
		wbot.SetMaxBodySize(1024 * 1024),
		wbot.SetUserAgents([]string{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"}),
	}

	// new bot
	bot, err := wbot.NewWBot(opts...)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer bot.Close()

	// stream
	go func() {
		count := 0
		for resp := range bot.Stream() {
			count++
			fmt.Printf("num: %d - depth: %d - visited url:%s - status:%d - body len: %d\n", count, resp.Depth, resp.URL.String(), resp.Status, len(resp.Body))
		}
	}()

	site := "https://www.github.com"

	if err := bot.Crawl(site); err != nil {
		log.Fatal(err)
	}

	fmt.Println("i'm out :)")
}
```

### TODO
- [ ] Add support for robots.txt.
- [ ] Add test cases.
- [ ] Implement `Fetch` using Chromedp.
- [ ] Add more examples.

### Bugs
Bugs or suggestions? Please visit the [issue tracker](https://github.com/twiny/wbot/issues).
