package wbot

// Logger
type Logger interface {
	Send(rep *Report)
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
func newReport(resp *Response, err error) *Report {
	var (
		rurl         = ""
		status       = 0
		depth  int32 = 0
	)

	if resp != nil {
		rurl = resp.URL.String()
		status = resp.Status
		depth = resp.Depth
	}

	return &Report{
		RequestURL: rurl,
		Status:     status,
		Depth:      depth,
		Err:        err,
	}
}
