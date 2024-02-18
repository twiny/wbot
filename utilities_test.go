package wbot

import "testing"

func TestHostname(t *testing.T) {
	validURLs := []struct {
		input    string
		expected string
	}{
		{"http://www.google.com", "google.com"},
		{"https://sub.domain.google.com", "google.com"},
		{"http://beta.moon.facebook.com", "facebook.com"},
		// ... Add more valid test cases here
	}

	invalidURLs := []string{
		"http://www.google.invalidTLD",
		"https://example.com.xxy",
		"ftp://example.site", // assuming "site" is not in your TLDs map
		// ... Add more invalid test cases here
	}

	for _, tt := range validURLs {
		got, err := Hostname(tt.input)
		if err != nil {
			t.Errorf("Hostname(%q) returned unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("Hostname(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}

	for _, url := range invalidURLs {
		_, err := Hostname(url)
		if err == nil {
			t.Errorf("Hostname(%q) expected to return an error, but got none", url)
		}
	}
}
