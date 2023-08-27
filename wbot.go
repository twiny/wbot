package wbot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/weppos/publicsuffix-go/publicsuffix"
)

type (
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
		Cancel()
		IsDone() bool
		Close() error
	}

	Logger interface {
		Write(ctx context.Context, log *Log) error
		Close() error
	}

	MetricsMonitor interface {
		IncTotalRequests()
		IncSuccessfulRequests()
		IncFailedRequests()
		IncRetries()
		IncRedirects()

		IncTotalPages()
		IncCrawledPages()
		IncSkippedPages()
		IncParsedLinks()

		IncClientErrors()
		IncServerErrors()
	}

	Request struct {
		Target *ParsedURL `json:"target"`
		Param  *Param     `json:"param"`
		Depth  int32      `json:"depth"`
	}

	Response struct {
		URL         *ParsedURL    `json:"url"`
		Status      int           `json:"status"`
		Body        []byte        `json:"-"`
		NextURLs    []*ParsedURL  `json:"next_urls"`
		Depth       int32         `json:"depth"`
		ElapsedTime time.Duration `json:"elapsed_time"`
		Err         error         `json:"-"`
	}

	ParsedURL struct {
		Hash string   `json:"hash"`
		Root string   `json:"root"`
		URL  *url.URL `json:"url"`
	}

	Param struct {
		Proxy       string `json:"proxy"`
		UserAgent   string `json:"user_agent"`
		Referer     string `json:"referer"`
		MaxBodySize int64  `json:"max_body_size"`
	}

	FilterRule struct {
		Hostname string           `json:"hostname"`
		Allow    []*regexp.Regexp `json:"allow"`
		Disallow []*regexp.Regexp `json:"disallow"`
	}

	RateLimit struct {
		Hostname string `json:"hostname"`
		Rate     string `json:"rate"`
	}

	Log struct {
		RequestURL   string        `json:"request_url"`
		Status       int           `json:"status"`
		Depth        int32         `json:"depth"`
		Err          error         `json:"err"`
		Timestamp    time.Time     `json:"timestamp"`
		ResponseTime time.Duration `json:"response_time"`
		ContentSize  int64         `json:"content_size"`
		UserAgent    string        `json:"user_agent"`
		RedirectURL  string        `json:"redirect_url"`
	}
)

func (r *Request) ResolveURL(u string) (*url.URL, error) {
	if strings.HasPrefix(u, "#") {
		return nil, fmt.Errorf("url is a fragment")
	}

	absURL, err := r.Target.URL.Parse(u)
	if err != nil {
		return nil, err
	}

	absURL.Fragment = ""

	return absURL, nil
}

func NewURL(raw string) (*ParsedURL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	domain, err := publicsuffix.Domain(u.Hostname())
	if err != nil {
		return nil, fmt.Errorf("failed to extract domain: %w", err)
	}

	// Ensure that the extracted TLD is in our allowed list
	tld := domain[strings.LastIndex(domain, ".")+1:]
	if !tlds[tld] {
		return nil, fmt.Errorf("invalid TLD: %s", tld)
	}

	hash, err := hashLink(*u)
	if err != nil {
		return nil, fmt.Errorf("invalid hash: %s", hash)
	}

	return &ParsedURL{
		Hash: hash,
		Root: domain,
		URL:  u,
	}, nil
}

func FindLinks(body []byte) (hrefs []string) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return hrefs
	}

	doc.Find("a[href]").Each(func(index int, item *goquery.Selection) {
		if href, found := item.Attr("href"); found {
			hrefs = append(hrefs, href)
		}
	})
	doc.Find("link[href]").Each(func(index int, item *goquery.Selection) {
		if href, found := item.Attr("href"); found {
			hrefs = append(hrefs, href)
		}
	})
	doc.Find("img[src]").Each(func(index int, item *goquery.Selection) {
		if src, found := item.Attr("src"); found {
			hrefs = append(hrefs, src)
		}
	})
	doc.Find("script[src]").Each(func(index int, item *goquery.Selection) {
		if src, found := item.Attr("src"); found {
			hrefs = append(hrefs, src)
		}
	})
	doc.Find("iframe[src]").Each(func(index int, item *goquery.Selection) {
		if src, found := item.Attr("src"); found {
			hrefs = append(hrefs, src)
		}
	})
	return hrefs
}

func hashLink(parsedLink url.URL) (string, error) {
	parsedLink.Scheme = ""

	parsedLink.Host = strings.TrimPrefix(parsedLink.Host, "www.")

	decodedPath, err := url.PathUnescape(parsedLink.Path)
	if err != nil {
		return "", err
	}
	parsedLink.Path = decodedPath

	cleanedURL := strings.TrimRight(parsedLink.String(), "/")

	cleanedURL = strings.TrimPrefix(cleanedURL, "//")

	hasher := sha256.New()
	hasher.Write([]byte(cleanedURL))

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
