package wbot

import (
	"net/url"
	"regexp"
)

var (
	badExtensions = regexp.MustCompile(`^.*\.(png|jpg|jpeg|gif|ico|eps|pdf|iso|mp3|mp4|zip|aif|mpa|wav|wma|7z|deb|pkg|rar|rpm|bin|dmg|dat|tar|exe|ps|psd|svg|tif|tiff|pps|ppt|pptx|xls|xlsx|wmv|doc|docx|txt|mov|mpl)$`)
)

//
// Filter
type filter struct {
	allowed    []*regexp.Regexp
	disallowed []*regexp.Regexp
}

// newFilter
func newFilter(allowed, disallowed []string) *filter {
	var f = &filter{
		allowed:    make([]*regexp.Regexp, 0),
		disallowed: make([]*regexp.Regexp, 0),
	}

	for _, p := range allowed {
		f.allowed = append(f.allowed, regexp.MustCompile(p))
	}

	for _, p := range disallowed {
		f.disallowed = append(f.disallowed, regexp.MustCompile(p))
	}

	return f
}

// Allow
func (f *filter) Allow(l *url.URL) bool {
	raw := l.String()

	if badExtensions.MatchString(l.Path) {
		return false
	}

	// disallowed
	for _, d := range f.disallowed {
		if d.MatchString(raw) {
			return false
		}
	}

	// allowed
	for _, a := range f.allowed {
		if !a.MatchString(raw) {
			return false
		}
	}

	return true
}
