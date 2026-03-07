package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

func poster[T any](title string, subTitle schwifty.Widgetable[T], coverURL string) schwifty.Button {
	return Button().
		Child(
			VStack(
				Picture().
					SizeRequest(180, 270).
					FromPaintable(gdk.NewTextureFromResource("/dev/skillless/Scanline/icons/scalable/state/missing-album.svg")).
					ConnectRealize(func(w gtk.Widget) {
						if preference.Performance().AllowPreviewImages() {
							imageutils.LoadIntoPictureScaled(coverURL, 180, 270, gtk.PictureNewFromInternalPtr(w.Ptr))
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
