package imageutils

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"codeberg.org/dergs/tonearm/pkg/schwifty/tracking"
	"codeberg.org/dergs/tonearm/pkg/schwifty/utils/weak"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

func LoadIntoImage(url string, image *gtk.Image) {
	ref := weak.NewWidgetRef(&image.Widget)
	go func() {
		texture, err := Load(url)
		if err != nil {
			return
		}
		tracking.SetFinalizer("Texture", texture)

		schwifty.OnMainThreadOnce(func(u uintptr) {
			ref.Use(func(widget *gtk.Widget) {
				gtk.ImageNewFromInternalPtr(widget.GoPointer()).SetFromPaintable(texture)
			})
			texture = nil
		}, 0)
	}()
}

func LoadIntoImageCropped(url string, size int, image *gtk.Image) {
	ref := weak.NewWidgetRef(&image.Widget)
	go func() {
		texture, err := Load(url)
		if err != nil {
			return
		}
		cropped := Crop(texture)
		texture.Unref()

		scaled := Scale(cropped, size, size)
		cropped.Unref()
		tracking.SetFinalizer("Texture", scaled)

		schwifty.OnMainThreadOnce(func(u uintptr) {
			ref.Use(func(widget *gtk.Widget) {
				gtk.ImageNewFromInternalPtr(widget.GoPointer()).SetFromPaintable(scaled)
			})
			scaled = nil
		}, 0)
	}()
}

func LoadIntoImageScaled(url string, width, height int, image *gtk.Image) {
	ref := weak.NewWidgetRef(&image.Widget)
	go func() {
		texture, err := Load(url)
		if err != nil {
			return
		}

		scaled := Scale(texture, width, height)
		texture.Unref()
		tracking.SetFinalizer("Texture", scaled)

		schwifty.OnMainThreadOnce(func(u uintptr) {
			ref.Use(func(widget *gtk.Widget) {
				gtk.ImageNewFromInternalPtr(widget.GoPointer()).SetFromPaintable(scaled)
			})
			scaled = nil
		}, 0)
	}()
}

func LoadIntoPictureCropped(url string, size int, picture *gtk.Picture) {
	ref := weak.NewWidgetRef(&picture.Widget)
	go func() {
		texture, err := Load(url)
		if err != nil {
			return
		}
		cropped := Crop(texture)
		texture.Unref()

		scaled := Scale(cropped, size, size)
		cropped.Unref()
		tracking.SetFinalizer("Texture", scaled)

		schwifty.OnMainThreadOnce(func(u uintptr) {
			ref.Use(func(widget *gtk.Widget) {
				gtk.PictureNewFromInternalPtr(widget.GoPointer()).SetPaintable(scaled)
			})
			scaled = nil
		}, 0)
	}()
}

func LoadIntoPictureScaled(url string, width, height int, picture *gtk.Picture) {
	ref := weak.NewWidgetRef(&picture.Widget)
	go func() {
		texture, err := Load(url)
		if err != nil {
			return
		}

		scaled := Scale(texture, width, height)
		texture.Unref()
		tracking.SetFinalizer("Texture", scaled)

		schwifty.OnMainThreadOnce(func(u uintptr) {
			ref.Use(func(widget *gtk.Widget) {
				gtk.PictureNewFromInternalPtr(widget.GoPointer()).SetPaintable(scaled)
			})
			scaled = nil
		}, 0)
	}()
}
