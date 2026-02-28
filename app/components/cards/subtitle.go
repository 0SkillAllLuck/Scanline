package cards

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/jwijenbergh/puregotk/v4/gtk"
	"github.com/jwijenbergh/puregotk/v4/pango"
)

func subTitle(text string) schwifty.Label {
	return Label(text).
		FontWeight(400).
		MaxWidthChars(20).
		WithCSSClass("dimmed").
		HAlign(gtk.AlignStartValue).
		Ellipsis(pango.EllipsizeEndValue)
}
