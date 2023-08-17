package crawler

import (
	"github.com/temoto/robotstxt"
)

const (
	robotstxtPath = "/robots.txt"
)

type (
	robortManager struct {
		robots map[string]*robotstxt.RobotsData
	}
)

func NewRobotManager() *robortManager {
	return &robortManager{
		robots: make(map[string]*robotstxt.RobotsData),
	}
}

// func (rm *robortManager) AddRobotsTxt(hostname string, statusCode int, body []byte) error {
// }

// func (rm *robortManager) Allowed(userAgent, path string) bool {
// }
