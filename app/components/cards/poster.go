package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/resources"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/imageutils"
	"github.com/jwijenbergh/puregotk/v4/gtk"
	"github.com/jwijenbergh/puregotk/v4/pango"
)

func poster[T any](title string, subTitle schwifty.Widgetable[T], coverUrl string) schwifty.Button {
	return Button().
		Child(
			VStack(
				Picture().
					SizeRequest(180, 270).
					FromPaintable(resources.MissingAlbum()).
					ConnectRealize(func(w gtk.Widget) {
						if preference.Performance().AllowPreviewImages() {
							imageutils.LoadIntoPictureScaled(coverUrl, 180, 270, gtk.PictureNewFromInternalPtr(w.Ptr))
						}
					}).
					CornerRadius(10).Overflow(gtk.OverflowHiddenValue),
				Label(title).
					WithCSSClass("heading").
					MarginTop(10).
					MaxWidthChars(18).
					HAlign(gtk.AlignStartValue).
					Ellipsis(pango.EllipsizeEndValue),
				subTitle.MarginTop(4),
			),
		).
		Padding(15).
		HExpand(false).
		VExpand(false).
		WithCSSClass("flat")
}
