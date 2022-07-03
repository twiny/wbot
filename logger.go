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
	requestURL := ""
	if resp.URL != nil {
		requestURL = resp.URL.String()
	}
	//
	return Report{
		RequestURL: requestURL,
		Status:     resp.Status,
		Depth:      resp.Depth,
		Err:        err,
	}
}
