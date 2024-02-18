package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/weppos/publicsuffix-go/publicsuffix"
)

var (
	//go:embed tlds.json
	tldsBytes []byte
	tlds      = map[string]bool{}
	once      = &sync.Once{}
)

func init() {
	once.Do(func() {
		tlds = make(map[string]bool)
		if err := json.Unmarshal(tldsBytes, &tlds); err != nil {
			return
		}
	})
}

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

	Request struct {
		Target *ParsedURL
		Param  *Param
		Depth  int32
	}

	Response struct {
		URL         *ParsedURL
		Status      int
		Body        []byte
		NextURLs    []*ParsedURL
		Depth       int32
		ElapsedTime time.Duration
		Err         error
	}

	ParsedURL struct {
		Hash string
		Root string
		URL  *url.URL
	}

	Param struct {
		Proxy       string
		UserAgent   string
		Referer     string
		MaxBodySize int64
		Timeout     time.Duration
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
func (u *ParsedURL) String() string {
	var link = u.URL.String()
	if len(link) > 64 {
		return link[:64]
	}
	return link
}

func NewURL(raw string) (*ParsedURL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	// Extract domain and TLD using publicsuffix-go
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
func Hostname(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Extract domain and TLD using publicsuffix-go
	domain, err := publicsuffix.Domain(u.Hostname())
	if err != nil {
		return "", fmt.Errorf("failed to extract domain: %w", err)
	}

	// Ensure that the extracted TLD is in our allowed list
	tld := domain[strings.LastIndex(domain, ".")+1:]
	if !tlds[tld] {
		return "", fmt.Errorf("invalid TLD: %s", tld)
	}

	return domain, nil
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
