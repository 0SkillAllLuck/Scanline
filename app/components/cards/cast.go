package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

func NewCastMember(name, role, thumbURL string) schwifty.Box {
	children := []any{
		Bin().
			Child(
				Picture().
					SizeRequest(140, 140).
					ContentFit(gtk.ContentFitCoverValue).
					ConnectRealize(func(w gtk.Widget) {
						if preference.Performance().AllowPreviewImages() && thumbURL != "" {
							imageutils.LoadIntoPictureCropped(thumbURL, 140, gtk.PictureNewFromInternalPtr(w.Ptr))
						}
					}),
			).
			SizeRequest(140, 140).
			HExpand(false).
			VExpand(false).
			CornerRadius(70).
			Overflow(gtk.OverflowHiddenValue),
		Label(name).
			WithCSSClass("heading").
			MarginTop(8).
			MaxWidthChars(18).
			HAlign(gtk.AlignCenterValue).
			Ellipsis(pango.EllipsizeEndValue),
	}

	if role != "" {
		children = append(children, Label(role).
			WithCSSClass("dimmed").
			MarginTop(2).
			MaxWidthChars(18).
			HAlign(gtk.AlignCenterValue).
			Ellipsis(pango.EllipsizeEndValue))
	}

	return VStack(children...).
		HAlign(gtk.AlignCenterValue).
		HExpand(false).
		VExpand(false).
		Padding(10)
}
