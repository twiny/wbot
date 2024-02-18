# Wiki WBot

A configurable, thread-safe web crawler, provides a minimal interface for crawling and downloading web pages.

- API
  - [Functions](#functions)
  - [Interfaces](#interfaces)
  
- Examples
  - [Install](#install)
  - example: [default options](#using-default-options)

## API

the core structure and interfaces of WBot are defined in the `pkg/api` package.

### Functions

#### WBot

```go
 New(opts ...Option) *WBot

// ...

 Run(links ...string) error
 OnReponse(fn func(*wbot.Response))
 Metrics() map[string]int64
 Shutdown()
```

#### Methods

```go
(r *Request) ResolveURL(u string) (*url.URL, error)
(u *ParsedURL) String() string
```

#### Common Functions

```go
FindLinks(body []byte) (hrefs []string)
NewURL(raw string) (*ParsedURL, error)
Hostname(link string) (string, error)
```

### Interfaces

```go
 Fetcher interface {
  Fetch(ctx context.Context, req *Request) (*Response, error)
  Close() error
 }

 Store interface {
  HasVisited(ctx context.Context, u *ParsedURL) (bool, error)
  Close() error
 }

 Queue interface {
  Push(ctx context.Context, req *Request) error
  Pop(ctx context.Context) (*Request, error)
  Len() int32
  Close() error
 }

 MetricsMonitor interface {
  IncTotalRequests()
  IncSuccessfulRequests()
  IncFailedRequests()

  IncTotalLink()
  IncCrawledLink()
  IncSkippedLink()
  IncDuplicatedLink()

  Metrics() map[string]int64
 }
```

## Example

### Install

requires Go1.22

`go get github.com/twiny/wbot`

### Using default options

```go
package main

import (
 "fmt"
 "log"

 "github.com/rs/zerolog"

 "github.com/twiny/wbot"
 "github.com/twiny/wbot/pkg/api"
)

func main() {
 bot := wbot.New(
  wbot.WithParallel(50),
  wbot.WithMaxDepth(5),
  wbot.WithRateLimit(&api.RateLimit{
   Hostname: "*",
   Rate:     "10/1s",
  }),
  wbot.WithLogLevel(zerolog.DebugLevel),
 )
 defer bot.Shutdown()

 // read responses
 bot.OnReponse(func(resp *api.Response) {
  fmt.Printf("crawled: %s\n", resp.URL.String())
 })

 if err := bot.Run(
  "https://go.dev/",
 ); err != nil {
  log.Fatal(err)
 }

 log.Printf("finished crawling\n")
}
```
