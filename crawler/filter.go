package crawler

import (
	"net/url"
	"regexp"

	"github.com/twiny/wbot"
)

var (
	badExtensions = regexp.MustCompile(`\.(png|jpg|jpeg|gif|ico|eps|pdf|iso|mp3|mp4|zip|aif|mpa|wav|wma|7z|deb|pkg|rar|rpm|bin|dmg|dat|tar|exe|ps|psd|svg|tif|tiff|pps|ppt|pptx|xls|xlsx|wmv|doc|docx|txt|mov|mpl)$`)
)

type (
	filter struct {
		rules map[string]*wbot.FilterRule
	}
)

func newFilter(rules ...*wbot.FilterRule) *filter {
	f := &filter{
		rules: make(map[string]*wbot.FilterRule),
	}

	for _, rule := range rules {
		f.rules[rule.Hostname] = rule
	}

	return f
}

func (f *filter) allow(link *url.URL) bool {
	hostname, err := wbot.Hostname(link.String())
	if err != nil {
		// review: double check this case
	}

	if badExtensions.MatchString(link.String()) {
		return false
	}

	rule, found := f.rules[hostname]
	if !found {
		// check if there is a wildcard rule
		rule, found = f.rules["*"]
		if !found {
			return true
		}
	}

	for _, pattern := range rule.Disallow {
		if pattern.MatchString(link.String()) {
			return false
		}
	}

	for _, pattern := range rule.Allow {
		if pattern.MatchString(link.String()) {
			return true
		}
	}

	return false // review: default deny
}
