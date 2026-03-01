package main

import (
	_ "embed"
	"log"
	"log/slog"
	"os"

	_ "github.com/0skillallluck/scanline/internal/features/macosfixes"

	"codeberg.org/dergs/tonearm/pkg/schwifty/tracking"
	"github.com/0skillallluck/scanline/app"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gio"
	"github.com/jwijenbergh/puregotk/v4/glib"
)

//go:generate glib-compile-schemas ./assets/meta
//go:generate glib-compile-resources --sourcedir=./assets/icons --target=./assets/meta/icons.gresource ./assets/meta/icons.gresource.xml
//go:generate scss ./assets/styles/style.scss ./assets/styles/style.css
//go:generate glib-compile-resources --sourcedir=./assets/styles --target=./assets/meta/styles.gresource ./assets/styles/styles.gresource.xml

//go:embed assets/meta/icons.gresource
var IconResources []byte

//go:embed assets/meta/styles.gresource
var StyleResources []byte

func init() {
}
func init() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	if os.Getenv("SCANLINE_DEBUG") == "1" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		go tracking.LogAliveObjects()
	}

	// Register resources
	iconResources, err := gio.NewResourceFromData(glib.NewBytes(IconResources, uint(len(IconResources))))
	if err != nil {
		log.Panicln("Failed to create resources: ", err)
	}
	gio.ResourcesRegister(iconResources)
	styleResources, err := gio.NewResourceFromData(glib.NewBytes(StyleResources, uint(len(StyleResources))))
	if err != nil {
		log.Panicln("Failed to create resources: ", err)
	}
	gio.ResourcesRegister(styleResources)
}

func main() {
	application := adw.NewApplication("dev.skillless.Scanline", gio.GApplicationDefaultFlagsValue)
	defer application.Unref()
	application.ConnectActivate(new(app.OnActivate(application)))

	if code := application.Run(len(os.Args), os.Args); code > 0 {
		application.Quit()
		os.Exit(code)
	}
}
