package router

import (
	"github.com/0skillallluck/scanline/internal/gettext"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
)

func errorHandler(err error) *Response {
	return &Response{
		PageTitle: gettext.Get("Internal Error"),
		View: StatusPage().
			Title(gettext.Get("Internal Error")).
			Description(gettext.Get("Unfortunately an error occurred while loading this view. Please try again later. If the error persists, please open an issue!\n\nError Message: ") + err.Error()).
			IconName("sentiment-dissatisfied-symbolic"),
	}
}
