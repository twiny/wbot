package fetcher

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/twiny/wbot"
)

type (
	defaultHTTPClient struct {
		client *http.Client
	}
)

func NewHTTPClient() wbot.Fetcher {
	return &defaultHTTPClient{
		client: &http.Client{
			Jar:     http.DefaultClient.Jar,
			Timeout: 30 * time.Second,
		},
	}
}

func (f *defaultHTTPClient) Fetch(ctx context.Context, req *wbot.Request) (*wbot.Response, error) {
	type (
		fetchResult struct {
			result *wbot.Response
			err    error
		}
	)

	var ch = make(chan fetchResult, 1)

	go func() {
		resp, err := f.fetch(req)
		if err != nil {
			ch <- fetchResult{nil, err}
			return
		}
		ch <- fetchResult{resp, nil}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case resp := <-ch:
			if resp.err != nil {
				return nil, resp.err
			}
			return resp.result, nil
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
		URL:        req.URL,
		Header:     header,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	})
	if err != nil {
		return nil, err
	}

	// Limit response body reading
	bodyReader := io.LimitReader(resp.Body, req.Param.MaxBodySize)

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	return &wbot.Response{
		URL:      req.URL,
		Status:   resp.StatusCode,
		Body:     body,
		NextURLs: wbot.FindLinks(body),
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
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100, // Default: 100
		MaxIdleConnsPerHost:   2,   // Default: 2
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// DisableKeepAlives:     true,
	}
}
