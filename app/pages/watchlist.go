package pages

import (
	"context"
	"strconv"

	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/adw"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

var WatchlistRoute = router.NewRoute("watchlist", watchlist)

func watchlist(ctx context.Context, appCtx *appctx.AppContext) *router.Response {
	items, err := appCtx.Manager.Watchlist(ctx)
	if err != nil {
		return router.FromError(gettext.Get("Watchlist"), err)
	}

	if len(items) == 0 {
		return &router.Response{
			PageTitle: gettext.Get("Watchlist"),
			View: StatusPage().
				IconName("starred-symbolic").
				Title(gettext.Get("Watchlist")).
				Description(gettext.Get("Your watchlist is empty.")),
		}
	}

	matches := appCtx.Manager.ResolveWatchlist(ctx, items)

	body := WrapBox().
		ConnectConstruct(func(w *adw.WrapBox) {
			w.SetChildSpacing(20)
			w.SetLineSpacing(20)
			w.SetLineHomogeneous(true)
			w.SetJustify(adw.JustifyFillValue)
		})

	for _, item := range items {
		var subtitle string
		switch item.Type {
		case "movie":
			if item.Year > 0 {
				subtitle = strconv.Itoa(item.Year)
			}
		case "show":
			subtitle = gettext.Get("TV Show")
		default:
			if item.Year > 0 {
				subtitle = strconv.Itoa(item.Year)
			}
		}

		var match *sources.WatchlistMatch
		if m, ok := matches[item.GUID]; ok {
			match = &m
		}

		body = body.Append(watchlistPoster(item.Title, subtitle, item.Thumb, match))
	}

	return &router.Response{
		PageTitle: gettext.Get("Watchlist"),
		View: ScrolledWindow().
			Child(body.VMargin(20).HMargin(20)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}

func watchlistPoster(title, subtitle, thumbURL string, match *sources.WatchlistMatch) any {
	btn := Button().
		Child(
			VStack(
				Picture().
					SizeRequest(180, 270).
					FromPaintable(gdk.NewTextureFromResource("/dev/skillless/Scanline/icons/scalable/state/missing-album.svg")).
					ConnectRealize(func(w gtk.Widget) {
						if thumbURL != "" && preference.Performance().AllowPreviewImages() {
							imageutils.LoadIntoPictureScaled(thumbURL, 180, 270, gtk.PictureNewFromInternalPtr(w.Ptr))
						}
					}).
					CornerRadius(10).
					Overflow(gtk.OverflowHiddenValue),
				Label(title).
					WithCSSClass("heading").
					MarginTop(10).
					MaxWidthChars(18).
					HAlign(gtk.AlignStartValue).
					Ellipsis(pango.EllipsizeEndValue),
				Label(subtitle).
					WithCSSClass("dim-label").
					WithCSSClass("caption").
					MarginTop(4).
					MaxWidthChars(18).
					HAlign(gtk.AlignStartValue).
					Ellipsis(pango.EllipsizeEndValue),
			),
		).
		Padding(15).
		HExpand(false).
		VExpand(false).
		WithCSSClass("flat")

	if match != nil {
		switch match.Type {
		case "movie":
			btn = btn.
				ActionName("win.route.movie").
				ActionTargetValue(glib.NewVariantString(match.ServerID + "/" + match.RatingKey))
		case "show":
			btn = btn.
				ActionName("win.route.show").
				ActionTargetValue(glib.NewVariantString(match.ServerID + "/" + match.RatingKey))
		}
	}

	return btn
}
