package imageutils

import (
	"math"

	"github.com/jwijenbergh/puregotk/v4/gdk"
	"github.com/jwijenbergh/puregotk/v4/gdkpixbuf"
)

func Crop(texture *gdk.Texture) *gdk.Texture {
	texture.Ref()
	defer texture.Unref()

	size := int(math.Min(float64(texture.GetIntrinsicWidth()), float64(texture.GetIntrinsicHeight())))
	src_x := (texture.GetIntrinsicWidth() - size) / 2
	src_y := (texture.GetIntrinsicHeight() - size) / 2

	pixbuf := gdk.PixbufGetFromTexture(texture)
	defer pixbuf.Unref()

	cropped := pixbuf.NewSubpixbuf(src_x, src_y, size, size)
	defer cropped.Unref()

	return gdk.NewTextureForPixbuf(cropped)
}

func Scale(texture *gdk.Texture, targetW, targetH int) *gdk.Texture {
	texture.Ref()
	defer texture.Unref()

	pixbuf := gdk.PixbufGetFromTexture(texture)
	defer pixbuf.Unref()

	scaled := pixbuf.ScaleSimple(targetW, targetH, gdkpixbuf.GdkInterpBilinearValue)
	defer scaled.Unref()

	return gdk.NewTextureForPixbuf(scaled)
}
