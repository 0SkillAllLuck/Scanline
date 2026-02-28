package icons

import (
	_ "embed"
	"log"

	"github.com/jwijenbergh/puregotk/v4/gio"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

//go:generate glib-compile-resources icons.gresource.xml

//go:embed icons.gresource
var Resources []byte

func init() {
	resources, err := gio.NewResourceFromData(glib.NewBytes(Resources, uint(len(Resources))))
	if err != nil {
		log.Panicln("Failed to create resources: ", err)
	}
	gio.ResourcesRegister(resources)
}
