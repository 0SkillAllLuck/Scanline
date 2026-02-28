package preference

import (
	"github.com/jwijenbergh/puregotk/v4/gio"
	"github.com/jwijenbergh/puregotk/v4/gobject"
)

type PerformanceSettings struct {
	settings *gio.Settings
}

// Images

func (p *PerformanceSettings) BindAllowPreviewImages(target *gobject.Object, property string) {
	p.settings.Bind("allow-preview-images", target, property, gio.GSettingsBindNoSensitivityValue)
}

func (p *PerformanceSettings) AllowPreviewImages() bool {
	return p.settings.GetBoolean("allow-preview-images")
}

func (p *PerformanceSettings) BindAllowPosterImages(target *gobject.Object, property string) {
	p.settings.Bind("allow-poster-images", target, property, gio.GSettingsBindNoSensitivityValue)
}

// Caching

func (p *PerformanceSettings) BindCacheImages(target *gobject.Object, property string) {
	p.settings.Bind("cache-images", target, property, gio.GSettingsBindNoSensitivityValue)
}

func (p *PerformanceSettings) ShouldCacheImages() bool {
	return p.settings.GetBoolean("cache-images")
}

func (p *PerformanceSettings) BindCacheLibraries(target *gobject.Object, property string) {
	p.settings.Bind("cache-libraries", target, property, gio.GSettingsBindNoSensitivityValue)
}

func (p *PerformanceSettings) BindCacheMetadata(target *gobject.Object, property string) {
	p.settings.Bind("cache-metadata", target, property, gio.GSettingsBindNoSensitivityValue)
}

// Navigation

func (p *PerformanceSettings) BindMaxRouterHistorySize(target *gobject.Object, property string) {
	p.settings.Bind("max-router-history-size", target, property, gio.GSettingsBindNoSensitivityValue)
}

func (p *PerformanceSettings) MaxRouterHistorySize() int {
	return p.settings.GetInt("max-router-history-size")
}
