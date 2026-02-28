package resources

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/jwijenbergh/puregotk/v4/gdk"
)

var MissingAlbum = func() schwifty.Paintable {
	image := gdk.NewTextureFromResource("/dev/skillless/Scanline/icons/scalable/state/missing-album.svg")
	return image
}
