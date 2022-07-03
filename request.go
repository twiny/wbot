package wbot

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// Request
type Request struct {
	BaseDomain string
	URL        *url.URL
	Depth      int32
	Param      Param
}

// param
type Param struct {
	Referer     string
	MaxBodySize int64
	UserAgent   string
	Proxy       string
}

// newRequest
func newRequest(raw string, depth int32, p Param) (Request, error) {
	// TODO: check if url is empty
	u, err := url.Parse(raw)
	if err != nil {
		return Request{}, err
	}

	baseDomain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
	if err != nil {
		return Request{}, err
	}

	return Request{
		BaseDomain: baseDomain,
		URL:        u,
		Depth:      depth,
		Param:      p,
	}, nil
}

// AbsURL
func (r *Request) AbsURL(u string) (*url.URL, error) {
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
