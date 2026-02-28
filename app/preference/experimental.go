package preference

import (
	"github.com/jwijenbergh/puregotk/v4/gio"
	"github.com/jwijenbergh/puregotk/v4/gobject"
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
