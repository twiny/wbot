package wbot

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Fetcher
type Fetcher interface {
	Fetch(req Request) (Response, error)
	Close() error
}

// Default Fetcher

//
var (
	defaultUserAgent = `wbot/0.1`
)

// Fetcher
type fetcher struct {
	cli *http.Client
}

// defaultFetcher
func defaultFetcher() *fetcher {
	return &fetcher{
		cli: newHTTPClient(),
	}
}

// Fetch
func (f *fetcher) Fetch(req Request) (Response, error) {
	var (
		userAgent   = defaultUserAgent
		maxBodySize = int64(1024 * 1024 * 10)
	)

	if req.param.userAgent != "" {
		userAgent = req.param.userAgent
	}

	if req.param.maxBodySize > 0 {
		maxBodySize = req.param.maxBodySize
	}

	// add headers
	var header = make(http.Header)
	header.Set("User-Agent", userAgent)
	header.Set("Referer", req.param.referer)

	f.cli.Transport = newHTTPTransport(req.param.proxy)

	resp, err := f.cli.Do(&http.Request{
		Method:     http.MethodGet,
		URL:        req.URL,
		Header:     header,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	})
	if err != nil {
		return Response{}, err
	}

	// Limit response body reading
	bodyReader := io.LimitReader(resp.Body, maxBodySize)

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return Response{}, err
	}

	nextURLs := findLinks(body)

	resp.Body.Close()

	return Response{
		URL:      req.URL,
		Status:   resp.StatusCode,
		Body:     body,
		NextURLs: nextURLs,
		Depth:    req.Depth,
	}, nil
}

// Close
func (f *fetcher) Close() error {
	f.cli.CloseIdleConnections()
	return nil
}

// newHTTPClient
func newHTTPClient() *http.Client {
	return &http.Client{
		Jar:     http.DefaultClient.Jar,
		Timeout: 5 * time.Second,
	}
}

// newHTTPTransport
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
		// DisableKeepAlives:     true, // twiny
	}
}

// linkFinder finds links in a response
func findLinks(body []byte) []string {
	var hrefs []string

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return hrefs
	}

	doc.Find("a[href]").Each(func(index int, item *goquery.Selection) {
		if href, found := item.Attr("href"); found {
			hrefs = append(hrefs, href)
		}
	})

	return hrefs
}
