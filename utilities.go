package wbot

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/weppos/publicsuffix-go/publicsuffix"
)

var tlds = map[string]bool{
	"ac": true, "ae": true, "aero": true, "af": true, "ag": true, "am": true,
	"as": true, "asia": true, "at": true, "au": true, "ax": true, "be": true,
	"bg": true, "bi": true, "biz": true, "bj": true, "br": true, "by": true,
	"ca": true, "cat": true, "cc": true, "cl": true, "cn": true, "co": true,
	"com": true, "coop": true, "cx": true, "de": true, "dk": true, "dm": true,
	"dz": true, "edu": true, "ee": true, "eu": true, "fi": true, "fo": true,
	"fr": true, "ge": true, "gl": true, "gov": true, "gs": true, "hk": true,
	"hr": true, "hu": true, "id": true, "ie": true, "in": true, "info": true,
	"int": true, "io": true, "ir": true, "is": true, "je": true, "jobs": true,
	"kg": true, "kr": true, "la": true, "lu": true, "lv": true, "ly": true,
	"ma": true, "md": true, "me": true, "mk": true, "mobi": true, "ms": true,
	"mu": true, "mx": true, "name": true, "net": true, "nf": true, "ng": true,
	"no": true, "nu": true, "nz": true, "org": true, "pl": true, "pr": true,
	"pro": true, "pw": true, "ro": true, "ru": true, "sc": true, "se": true,
	"sg": true, "sh": true, "si": true, "sk": true, "sm": true, "st": true,
	"so": true, "su": true, "tc": true, "tel": true, "tf": true, "th": true,
	"tk": true, "tl": true, "tm": true, "tn": true, "travel": true, "tw": true,
	"tv": true, "tz": true, "ua": true, "uk": true, "us": true, "uz": true,
	"vc": true, "ve": true, "vg": true, "ws": true, "xxx": true, "rs": true,
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
