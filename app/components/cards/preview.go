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

func previewCard[T any](title string, subTitle schwifty.Widgetable[T], artUrl string, progress float64) schwifty.Button {
	// Progress bar width based on progress
	progressWidth := int(480 * progress)

	// Create the picture widget
	picture := Picture().
		SizeRequest(480, 270).
		FromPaintable(resources.MissingAlbum()).
		ContentFit(gtk.ContentFitCoverValue).
		ConnectRealize(func(w gtk.Widget) {
			if preference.Performance().AllowPreviewImages() {
				imageutils.LoadIntoPictureScaled(artUrl, 480, 270, gtk.PictureNewFromInternalPtr(w.Ptr))
			}
		})

	// Create progress bar at the bottom
	progressBar := Box(gtk.OrientationHorizontalValue).
		SizeRequest(progressWidth, 4).
		VAlign(gtk.AlignEndValue).
		HAlign(gtk.AlignStartValue).
		CSS("box { background-color: @accent_bg_color; }")

	// Create overlay with picture as base and progress bar on top
	overlay := gtk.NewOverlay()
	overlay.SetChild(picture.ToGTK())
	overlay.AddOverlay(progressBar.ToGTK())

	// Wrap overlay in a Bin for corner radius
	imageContainer := Bin().
		Child(ManagedWidget(&overlay.Widget)).
		SizeRequest(480, 270).
		CornerRadius(10).
		Overflow(gtk.OverflowHiddenValue)

	return Button().
		Child(
			VStack(
				imageContainer,
				Label(title).
					WithCSSClass("heading").
					MarginTop(10).
					MaxWidthChars(40).
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
