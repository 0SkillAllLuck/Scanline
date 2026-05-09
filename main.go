package main

import (
	_ "embed"
	"log"
	"log/slog"
	"os"

	_ "github.com/0skillallluck/scanline/internal/features/macosfixes"

	"codeberg.org/dergs/tonearm/pkg/schwifty/tracking"
	"codeberg.org/puregotk/puregotk/v4/adw"
	"codeberg.org/puregotk/puregotk/v4/gio"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"github.com/0skillallluck/scanline/app"
)

//go:generate glib-compile-schemas ./assets/meta
//go:generate glib-compile-resources --sourcedir=./assets/icons --target=./assets/meta/icons.gresource ./assets/meta/icons.gresource.xml
//go:generate glib-compile-resources --sourcedir=./assets/icons-darwin --target=./assets/meta/icons-darwin.gresource ./assets/meta/icons-darwin.gresource.xml
//go:generate scss ./assets/styles/style.scss ./assets/styles/style.css
//go:generate glib-compile-resources --sourcedir=./assets/styles --target=./assets/meta/styles.gresource ./assets/meta/styles.gresource.xml

//go:embed assets/meta/styles.gresource
var StyleResources []byte

func init() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	if os.Getenv("SCANLINE_DEBUG") == "1" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		go tracking.LogAliveObjects()
	}

	for _, bundle := range IconResources {
		registerResource(bundle)
	}
	registerResource(StyleResources)
}

func registerResource(data []byte) {
	res, err := gio.NewResourceFromData(glib.NewBytes(data, uint(len(data))))
	if err != nil {
		log.Panicln("Failed to create resources: ", err)
	}
	gio.ResourcesRegister(res)
}

func main() {
	glib.SetApplicationName("Scanline")
	glib.SetPrgname("Scanline")

	application := adw.NewApplication("dev.skillless.Scanline", gio.GApplicationDefaultFlagsValue)
	defer application.Unref()
	application.ConnectActivate(new(app.OnActivate(application)))

	if code := application.Run(int32(len(os.Args)), os.Args); code > 0 {
		application.Quit()
		os.Exit(int(code))
	}
}
