package fetcher

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/twiny/wbot"
)

type (
	defaultHTTPClient struct {
		client     *http.Client
		bufferPool *sync.Pool
	}
)

func NewHTTPClient() wbot.Fetcher {
	var (
		fn = func() any {
			return new(bytes.Buffer)
		}
	)

	return &defaultHTTPClient{
		client: &http.Client{
			Jar:     http.DefaultClient.Jar,
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 10 * time.Second,
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100, // Default: 100
				MaxIdleConnsPerHost:   2,   // Default: 2
				IdleConnTimeout:       10 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				// DisableKeepAlives:     true,
			},
		},
		bufferPool: &sync.Pool{
			New: fn,
		},
	}
}

func (f *defaultHTTPClient) Fetch(ctx context.Context, req *wbot.Request) (*wbot.Response, error) {
	var (
		respCh   = make(chan *wbot.Response, 1)
		fetchErr error
	)

	fctx, done := context.WithTimeout(ctx, req.Param.Timeout)
	defer done()

	go func() {
		resp, err := f.fetch(req)
		if err != nil {
			fetchErr = err
			return
		}
		respCh <- resp
	}()

	for {
		select {
		case <-fctx.Done():
			return nil, fctx.Err()
		case resp := <-respCh:
			if fetchErr != nil {
				return nil, fetchErr
			}
			return resp, nil
		}
	}
}
func (f *defaultHTTPClient) Close() error {
	f.client.CloseIdleConnections()
	return nil
}

func (f *defaultHTTPClient) fetch(req *wbot.Request) (*wbot.Response, error) {
	var header = make(http.Header)
	header.Set("User-Agent", req.Param.UserAgent)
	header.Set("Referer", req.Param.Referer)

	if req.Param.Proxy != "" {
		f.client.Transport = newHTTPTransport(req.Param.Proxy)
	}

	resp, err := f.client.Do(&http.Request{
		Method:     http.MethodGet,
		URL:        req.Target.URL,
		Header:     header,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := f.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer f.bufferPool.Put(buf)

	// Limit response body reading
	if _, err := io.CopyN(buf, resp.Body, req.Param.MaxBodySize); err != nil && err != io.EOF {
		return nil, err
	}

	bytes := buf.Bytes()

	links := wbot.FindLinks(bytes)

	var nextURLs []*wbot.ParsedURL
	for _, link := range links {
		absURL, err := req.ResolveURL(link)
		if err != nil {
			continue
		}
		parsedURL, err := wbot.NewURL(absURL.String())
		if err != nil {
			continue
		}
		nextURLs = append(nextURLs, parsedURL)
	}

	return &wbot.Response{
		URL:      req.Target,
		Status:   resp.StatusCode,
		Body:     bytes,
		NextURLs: nextURLs,
		Depth:    req.Depth,
	}, nil
}
func newHTTPTransport(purl string) *http.Transport {
	var proxy = http.ProxyFromEnvironment

	if purl != "" {
		proxy = func(req *http.Request) (*url.URL, error) {
			return url.Parse(purl)
		}
	}
	return &http.Transport{
		Proxy: proxy,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          10, // Default: 100
		MaxIdleConnsPerHost:   5,  // Default: 2
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ExpectContinueTimeout: 2 * time.Second,
		DisableKeepAlives:     true,
	}
}
