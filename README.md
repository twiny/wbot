# WBot

WBot is a configurable, thread-safe web crawler written in Go. It offers a clean and minimal API for crawling and downloading web pages.

## Features:
📦 Clean Minimal API: Easy-to-use API that gets you up and running in no time.
⚙️ Configurable: MaxDepth, MaxBodySize, Rate Limit, Parrallelism, User Agent & Proxy rotation.
🚀 High Performance: Memory-efficient and designed for multi-threaded tasks.
🔌 Extensible: Provides built-in interfaces for Fetcher, Store, Queue, and Logger.

### [Examples & API](https://github.com/twiny/wbot/wiki)

## Configurations:

WBot can be configured using the following options:

```go
WithParallel(parallel int) Option
WithMaxDepth(maxDepth int32) Option
WithUserAgents(userAgents []string) Option
WithProxies(proxies []string) Option
WithRateLimit(rates ...*wbot.RateLimit) Option
WithFilter(rules ...*wbot.FilterRule) Option
WithFetcher(fetcher wbot.Fetcher) Option
WithStore(store wbot.Store) Option
WithLogger(logger wbot.Logger) Option
```

## WBot APIs:

You can interact with WBot using the following methods:

```go
Start(links ...string)
OnReponse(fn func(*wbot.Response))
OnError(fn func(err error))
Stats() map[string]any
Stop()
```

## Quick Start

```go
package main

import (
	"github.com/twiny/wbot"
	"github.com/twiny/wbot/crawler"
)

func main() {
	bot := crawler.New(
		crawler.WithParallel(10),
		crawler.WithMaxDepth(10),
	)

	bot.OnReponse(func(resp *wbot.Response) {
		_ = resp
	})

	bot.OnError(func(err error) {
		_ = err
	})

	bot.Start(
		"https://www.github.com/",
		"https://crawler-test.com/",
		"https://www.warriorforum.com/",
	)
}
```

## TODO
- [ ] Add support for robots.txt.
- [ ] Add test cases.
- [ ] Implement `Fetch` using Chromedp.
- [ ] Add more examples.
- [ ] Add documentation.

## Bugs
Bugs or suggestions? Please visit the [issue tracker](https://github.com/twiny/wbot/issues).