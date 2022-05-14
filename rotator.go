package wbot

import (
	"container/ring"
)

// rotator
type rotator struct {
	r *ring.Ring
}

// newRotator
func newRotator(s []string) *rotator {
	r := ring.New(len(s))
	for _, item := range s {
		r.Value = item
		r = r.Next()
	}
	return &rotator{
		r: r,
	}
}

// Next
func (r *rotator) next() string {
	if r == nil {
		return ""
	}

	val, ok := r.r.Value.(string)
	if !ok {
		return ""
	}

	// move
	r.r = r.r.Next()

	return val
}
