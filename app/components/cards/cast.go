package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/imageutils"
	"github.com/jwijenbergh/puregotk/v4/gtk"
	"github.com/jwijenbergh/puregotk/v4/pango"
)

func NewCastMember(name, role, thumbURL string) schwifty.Box {
	children := []any{
		Picture().
			SizeRequest(140, 140).
			ContentFit(gtk.ContentFitCoverValue).
			ConnectRealize(func(w gtk.Widget) {
				if preference.Performance().AllowPreviewImages() && thumbURL != "" {
					imageutils.LoadIntoPictureCropped(thumbURL, 140, gtk.PictureNewFromInternalPtr(w.Ptr))
				}
			}).
			CSS("picture { min-width: 140px; min-height: 140px; border-radius: 50%; }").
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
		Padding(10)
}
