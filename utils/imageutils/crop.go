package imageutils

import (
	"math"

	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/gdkpixbuf"
)

func Crop(texture *gdk.Texture) *gdk.Texture {
	texture.Ref()
	defer texture.Unref()

	size := int32(math.Min(float64(texture.GetIntrinsicWidth()), float64(texture.GetIntrinsicHeight())))
	srcX := (texture.GetIntrinsicWidth() - size) / 2
	srcY := (texture.GetIntrinsicHeight() - size) / 2

	pixbuf := gdk.PixbufGetFromTexture(texture)
	defer pixbuf.Unref()

	cropped := pixbuf.NewSubpixbuf(srcX, srcY, size, size)
	defer cropped.Unref()

	return gdk.NewTextureForPixbuf(cropped)
}

func Scale(texture *gdk.Texture, targetW, targetH int) *gdk.Texture {
	texture.Ref()
	defer texture.Unref()

	pixbuf := gdk.PixbufGetFromTexture(texture)
	defer pixbuf.Unref()

	scaled := pixbuf.ScaleSimple(int32(targetW), int32(targetH), gdkpixbuf.GdkInterpBilinearValue)
	defer scaled.Unref()

	return gdk.NewTextureForPixbuf(scaled)
}
