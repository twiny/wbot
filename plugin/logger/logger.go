package logger

import (
	"time"

	"github.com/twiny/flog/v2"
	"github.com/twiny/wbot"
)

type (
	defaultLogger struct {
		l *flog.Logger
	}
)

func NewFileLogger(prefix string) (*defaultLogger, error) {
	logger, err := flog.NewLogger(prefix, 10, 10)
	if err != nil {
		return nil, err
	}

	return &defaultLogger{
		l: logger,
	}, nil
}

func (l *defaultLogger) Write(log *wbot.Log) error {
	f := []flog.Field{
		flog.NewField("request_url", log.RequestURL),
		flog.NewField("status", log.Status),
		flog.NewField("depth", log.Depth),
		flog.NewField("timestamp", log.Timestamp.Format(time.RFC3339)),
		flog.NewField("response_time", log.ResponseTime.String()),
		flog.NewField("content_size", log.ContentSize),
	}

	if log.UserAgent != "" {
		f = append(f, flog.NewField("user_agent", log.UserAgent))
	}

	if log.RedirectURL != "" {
		f = append(f, flog.NewField("redirect_url", log.RedirectURL))
	}

	if log.Err != nil {
		f = append(f, flog.NewField("error", log.Err.Error()))
		l.l.Error(log.Err.Error(), f...)
		return nil
	}

	l.l.Info("page", f...)
	return nil
}
func (l *defaultLogger) Close() error {
	l.l.Close()
	return nil
}
