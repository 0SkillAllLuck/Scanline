package preferences

import (
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/adw"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/internal/gettext"
)

var experimentalPreferences = PreferencesPage(
	PreferencesGroup(
		SwitchRow().
			Title(gettext.Get("Enable Watchlist")).
			Subtitle(gettext.Get("Show the Watchlist tab in the navigation bar.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Experimental().BindEnableWatchlist(&sr.Object, "active")
			}),
		SwitchRow().
			Title(gettext.Get("Enable Windowed Player")).
			Subtitle(gettext.Get("Allow the player to open in a window instead of fullscreen.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Experimental().BindEnableWindowedPlayer(&sr.Object, "active")
			}),
		SwitchRow().
			Title(gettext.Get("Enable PiP support")).
			Subtitle(gettext.Get("Enable support for picture-in-picture mode.")),
		SwitchRow().
			Title(gettext.Get("Enable Jellyfin support")).
			Subtitle(gettext.Get("Enable support for Jellyfin servers.")),
		SwitchRow().
			Title(gettext.Get("Enable EMBY support")).
			Subtitle(gettext.Get("Enable support for EMBY servers.")),
	).Title(gettext.Get("Features")).Description(gettext.Get("Toggle experimental features. These may be incomplete or unstable.")),
).Title(gettext.Get("Experimental")).IconName("science-symbolic")
