package pages

import (
	"github.com/0skillallluck/scanline/app/appctx"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
)

var WatchlistRoute = router.NewRoute("watchlist", watchlist)

func watchlist(appCtx *appctx.AppContext) *router.Response {
	return &router.Response{
		PageTitle: gettext.Get("Watchlist"),
		View: StatusPage().
			IconName("starred-symbolic").
			Title(gettext.Get("Watchlist")).
			Description(gettext.Get("Work in progress.")),
	}
}
