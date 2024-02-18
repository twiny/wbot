package crawler

import (
	"github.com/temoto/robotstxt"
)

const (
	robotstxtPath = "/robots.txt"
)

type robotManager struct {
	followRobots bool
	robots       map[string]*robotstxt.RobotsData
}

func newRobotManager(follow bool) *robotManager {
	return &robotManager{
		followRobots: follow,
		robots:       make(map[string]*robotstxt.RobotsData),
	}
}

func (rm *robotManager) AddRobotsTxt(hostname string, body []byte) error {
	data, err := robotstxt.FromBytes(body)
	if err != nil {
		return err // Return the error if parsing fails.
	}

	rm.robots[hostname] = data
	return nil
}
func (rm *robotManager) Allowed(userAgent, url string) bool {
	hostname := url // Simplification; use proper URL parsing in production.

	robotsData, exists := rm.robots[hostname]
	if !exists {
		return true
	}

	return robotsData.TestAgent(url, userAgent)
}
