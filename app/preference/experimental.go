package preference

import (
	"codeberg.org/puregotk/puregotk/v4/gio"
	"codeberg.org/puregotk/puregotk/v4/gobject"
)

type ExperimentalSettings struct {
	settings *gio.Settings
}

func (e *ExperimentalSettings) BindEnableWatchlist(target *gobject.Object, property string) {
	e.settings.Bind("enable-watchlist", target, property, gio.GSettingsBindNoSensitivityValue)
}

func (e *ExperimentalSettings) EnableWatchlist() bool {
	return e.settings.GetBoolean("enable-watchlist")
}

func (e *ExperimentalSettings) OnEnableWatchlistChanged(callback func()) {
	e.settings.ConnectSignal("changed::enable-watchlist", new(callback))
}

func (e *ExperimentalSettings) BindEnableWindowedPlayer(target *gobject.Object, property string) {
	e.settings.Bind("enable-windowed-player", target, property, gio.GSettingsBindNoSensitivityValue)
}

func (e *ExperimentalSettings) EnableWindowedPlayer() bool {
	return e.settings.GetBoolean("enable-windowed-player")
}

func (e *ExperimentalSettings) OnEnableWindowedPlayerChanged(callback func()) {
	e.settings.ConnectSignal("changed::enable-windowed-player", new(callback))
}

func (e *ExperimentalSettings) BindStartInFullscreen(target *gobject.Object, property string) {
	e.settings.Bind("start-in-fullscreen", target, property, gio.GSettingsBindNoSensitivityValue)
}

func (e *ExperimentalSettings) StartInFullscreen() bool {
	return e.settings.GetBoolean("start-in-fullscreen")
}
