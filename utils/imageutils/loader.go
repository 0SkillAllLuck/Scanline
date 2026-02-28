package imageutils

import (
	"github.com/jwijenbergh/puregotk/v4/gdk"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

func Load(url string) (*gdk.Texture, error) {
	image, err := fetch(url)
	if err != nil {
		return nil, err
	}

	gBytes := glib.NewBytes(image, uint(len(image)))
	texture, err := gdk.NewTextureFromBytes(gBytes)
	gBytes.Unref()

	return texture, err
}

func LoadCropped(url string) (*gdk.Texture, error) {
	image, err := fetch(url)
	if err != nil {
		return nil, err
	}

	gBytes := glib.NewBytes(image, uint(len(image)))
	texture, err := gdk.NewTextureFromBytes(gBytes)
	gBytes.Unref()

	if err != nil {
		return nil, err
	}

	cropped := Crop(texture)
	texture.Unref()

	return cropped, err
}
