# WBot

A configurable, thread-safe web crawler, provides a minimal interface for crawling and downloading web pages.

## Features

- Clean minimal API.
- Configurable: MaxDepth, MaxBodySize, Rate Limit, Parrallelism,  User Agent & Proxy rotation.
- Memory-efficient, thread-safe.
- Provides built-in interface: Fetcher, Store, Queue & a Logger.

## API

WBot provides a minimal API for crawling web pages.

```go
 Run(links ...string) error
 OnReponse(fn func(*wbot.Response))
 Metrics() map[string]int64
 Shutdown()
```

## Usage

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

### Wiki

More documentation can be found in the [wiki](https://github.com/twiny/wbot/wiki).

### Bugs

Bugs or suggestions? Please visit the [issue tracker](https://github.com/twiny/wbot/issues).
