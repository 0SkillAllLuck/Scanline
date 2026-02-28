package preference

import (
	"runtime"

	"github.com/0skillallluck/scanline/internal/g"
	"github.com/jwijenbergh/puregotk/v4/gio"
)

//go:generate glib-compile-schemas .

var General = g.Lazy(func() *GeneralSettings {
	return &GeneralSettings{
		finalize(gio.NewSettings("dev.skillless.Scanline")),
	}
})

var Performance = g.Lazy(func() *PerformanceSettings {
	return &PerformanceSettings{
		finalize(gio.NewSettings("dev.skillless.Scanline.performance")),
	}
})

var Experimental = g.Lazy(func() *ExperimentalSettings {
	return &ExperimentalSettings{
		finalize(gio.NewSettings("dev.skillless.Scanline.experimental")),
	}
})

func finalize(settings *gio.Settings) *gio.Settings {
	runtime.SetFinalizer(settings, func(s *gio.Settings) {
		s.Unref()
	})
	return settings
}
