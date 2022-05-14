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
	param      param
}

// param
type param struct {
	referer     string
	maxBodySize int64
	userAgent   string
	proxy       string
}

// NewRequest
func NewRequest(raw string, depth int32, p param) (*Request, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	baseDomain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
	if err != nil {
		return nil, err
	}

	return &Request{
		BaseDomain: baseDomain,
		URL:        u,
		Depth:      depth,
		param:      p,
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
