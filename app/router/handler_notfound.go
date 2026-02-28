package router

import (
	"github.com/0skillallluck/scanline/internal/gettext"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
)

func notFoundHandler() *Response {
	return &Response{
		PageTitle: gettext.Get("Not Found"),
		View: StatusPage().
			Title(gettext.Get("Not found")).
			Description(gettext.Get("The requested deeplink did not have any available handlers.")).
			IconName("sentiment-dissatisfied-symbolic"),
	}
}
