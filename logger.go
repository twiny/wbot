package wbot

// Logger
type Logger interface {
	Send(rep Report)
	Close() error
}

// Report
type Report struct {
	RequestURL string
	Status     int
	Depth      int32
	Err        error
}

// newReport
func newReport(resp Response, err error) Report {
	return Report{
		RequestURL: resp.URL.String(),
		Status:     resp.Status,
		Depth:      resp.Depth,
		Err:        err,
	}
}
