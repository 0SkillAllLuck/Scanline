package router

import (
	"time"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
)

type Response struct {
	Error     error
	ExpiresAt *time.Time
	PageTitle string
	Toolbar   schwifty.BaseWidgetable
	View      schwifty.BaseWidgetable
}

func FromError(pageTitle string, err error) *Response {
	return &Response{
		Error:     err,
		PageTitle: pageTitle,
		Toolbar:   nil,
		View:      nil,
	}
}
