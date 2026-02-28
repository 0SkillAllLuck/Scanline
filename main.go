package main

import (
	"log/slog"
	"os"

	_ "github.com/0skillallluck/scanline/app/icons"
	_ "github.com/0skillallluck/scanline/app/styles"

	_ "github.com/0skillallluck/scanline/internal/features/macosfixes"

	"codeberg.org/dergs/tonearm/pkg/schwifty/tracking"
	"github.com/0skillallluck/scanline/app"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gio"
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	if os.Getenv("SCANLINE_DEBUG") == "1" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		go tracking.LogAliveObjects()
	}
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
