package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

func NewInfoCard(icon, title, subtitle string) schwifty.Box {
	return VStack(
		Image().FromIconName(icon).PixelSize(24).HAlign(gtk.AlignCenterValue),
		Label(title).
			WithCSSClass("heading").
			MarginTop(8).
			HAlign(gtk.AlignCenterValue),
		Label(subtitle).
			WithCSSClass("dimmed").
			HAlign(gtk.AlignCenterValue),
	).
		HAlign(gtk.AlignFillValue).
		HExpand(true).
		HPadding(10)
}
