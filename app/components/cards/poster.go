package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

// Poster card image dimensions. Pre-sized cover URLs (via PosterCoverURL)
// match these so the server returns ready-to-render JPEGs and no local
// scaling is needed.
const (
	PosterWidth  = 180
	PosterHeight = 270
)

// PosterCoverURL returns a thumb→URL builder that asks src for cover art
// at the poster's exact display dimensions. Pages building horizontal
// poster lists pass the returned closure into lists.RenderHub etc.
func PosterCoverURL(src sources.Source) func(thumb string) string {
	return func(thumb string) string {
		return src.PhotoTranscodeURL(thumb, PosterWidth, PosterHeight)
	}
}

func poster[T any](title string, subTitle schwifty.Widgetable[T], coverURL string) schwifty.Button {
	return posterWithProgress(title, subTitle, coverURL, 0)
}

func posterWithProgress[T any](title string, subTitle schwifty.Widgetable[T], coverURL string, progress float64) schwifty.Button {
	picture := Picture().
		SizeRequest(PosterWidth, PosterHeight).
		FromPaintable(gdk.NewTextureFromResource("/dev/skillless/Scanline/icons/scalable/state/missing-album.svg")).
		ConnectRealize(func(w gtk.Widget) {
			if preference.Performance().AllowPreviewImages() {
				imageutils.LoadIntoPictureScaled(coverURL, PosterWidth, PosterHeight, gtk.PictureNewFromInternalPtr(w.Ptr))
			}
		})

	var image any
	if progress > 0 {
		progressBar := Box(gtk.OrientationHorizontalValue).
			SizeRequest(int32(PosterWidth*progress), 4).
			VAlign(gtk.AlignEndValue).
			HAlign(gtk.AlignStartValue).
			CSS("box { background-color: @accent_bg_color; }")

		image = Bin().
			Child(Overlay(picture).AddOverlay(progressBar)).
			SizeRequest(PosterWidth, PosterHeight).
			CornerRadius(10).
			Overflow(gtk.OverflowHiddenValue)
	} else {
		image = picture.CornerRadius(10).Overflow(gtk.OverflowHiddenValue)
	}

	return Button().
		Child(
			VStack(
				image,
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
