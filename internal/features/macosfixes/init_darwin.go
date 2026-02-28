//go:build darwin

package macosfixes

import "github.com/jwijenbergh/puregotk/pkg/core"

func init() {
	// puregotk defaults to Linux SONAMEs; macOS requires dylib names.
	core.SetSharedLibraries("CAIRO", []string{"libcairo-gobject.2.dylib"})
	core.SetSharedLibraries("GLIB", []string{"libgobject-2.0.dylib", "libglib-2.0.dylib"})
	core.SetSharedLibraries("GMODULE", []string{"libgmodule-2.0.dylib"})
	core.SetSharedLibraries("GOBJECT", []string{"libgobject-2.0.dylib"})
	core.SetSharedLibraries("GIO", []string{"libgio-2.0.dylib"})
	core.SetSharedLibraries("GDKPIXBUF", []string{"libgdk_pixbuf-2.0.dylib"})
	core.SetSharedLibraries("GRAPHENE", []string{"libgraphene-1.0.dylib"})
	core.SetSharedLibraries("PANGO", []string{"libpango-1.0.dylib"})
	core.SetSharedLibraries("GDK", []string{"libgtk-4.1.dylib"})
	core.SetSharedLibraries("GSK", []string{"libgtk-4.1.dylib"})
	core.SetSharedLibraries("GTK", []string{"libgtk-4.1.dylib"})
	core.SetSharedLibraries("ADW", []string{"libadwaita-1.dylib"})
}
