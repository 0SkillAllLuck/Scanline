package preference

import (
	"codeberg.org/puregotk/puregotk/v4/gio"
)

type GeneralSettings struct {
	settings *gio.Settings
}

func (g *GeneralSettings) ShouldHideSecretServiceWarning() bool {
	return g.settings.GetBoolean("hide-secret-service-warning")
}

func (g *GeneralSettings) SetHideSecretServiceWarning(hide bool) bool {
	return g.settings.SetBoolean("hide-secret-service-warning", hide)
}

func (g *GeneralSettings) GetWindowHeight() int32 {
	return g.settings.GetInt("window-height")
}

func (g *GeneralSettings) SetWindowHeight(height int32) {
	g.settings.SetInt("window-height", height)
}

func (g *GeneralSettings) GetWindowWidth() int32 {
	return g.settings.GetInt("window-width")
}

func (g *GeneralSettings) SetWindowWidth(width int32) {
	g.settings.SetInt("window-width", width)
}
