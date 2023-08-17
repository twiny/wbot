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

	Queue interface {
		Push(ctx context.Context, req *Request) error
		Pop(ctx context.Context) (*Request, error)
		Close() error
	}

	Store interface {
		HasVisited(ctx context.Context, key string) (bool, error)
		Close() error
	}

	Logger interface {
		Write(ctx context.Context, log *Log) error
		Close() error
	}

	MetricsMonitor interface {
		IncrementTotalRequests()
		IncrementSuccessfulRequests()
		IncrementFailedRequests()
		IncrementRetries()
		IncrementRedirects()

		IncrementTotalPages()
		IncrementCrawledPages()
		IncrementSkippedPages()
		IncrementParsedLinks()

		IncrementClientErrors()
		IncrementServerErrors()
	}

	Request struct {
		ID       string
		BaseHost string
		URL      *url.URL
		Depth    int32
		Param    *Param
	}

	Response struct {
		URL         *url.URL
		Status      int
		Body        []byte
		NextURLs    []string
		Depth       int32
		ElapsedTime time.Duration
		Err         error
	}

	Param struct {
		Proxy       string
		UserAgent   string
		Referer     string
		MaxBodySize int64
	}

	FilterRule struct {
		Hostname string
		Allow    []*regexp.Regexp
		Disallow []*regexp.Regexp
	}

	RateLimit struct {
		Hostname string
		Rate     string
	}

	Log struct {
		RequestURL   string
		Status       int
		Depth        int32
		Err          error
		Timestamp    time.Time
		ResponseTime time.Duration
		ContentSize  int64
		UserAgent    string
		RedirectURL  string
	}
)

func (r *Request) ResolveURL(u string) (*url.URL, error) {
	if strings.HasPrefix(u, "#") {
		return nil, fmt.Errorf("url is a fragment")
	}

	absURL, err := r.URL.Parse(u)
	if err != nil {
		return nil, err
	}

	absURL.Fragment = ""

	return absURL, nil
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

	// ... Add other tags and attributes as necessary

	return hrefs
}

func HashLink(link string) (string, error) {
	parsedLink, err := url.Parse(link)
	if err != nil {
		return "", err
	}

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

func Hostname(link string) (string, error) {
	hostname, err := publicsuffix.Domain(link)
	if err != nil {
		return "", fmt.Errorf("failed to get domain: %w", err)
	}
	return hostname, nil
}

func ValidURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	return u, nil
}
