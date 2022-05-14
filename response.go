package wbot

import "net/url"

// Response
type Response struct {
	URL      *url.URL
	Status   int
	Body     []byte
	NextURLs []string
	Depth    int32
}
