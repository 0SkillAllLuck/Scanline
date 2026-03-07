package preferences

import (
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/adw"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/internal/gettext"
)

var playerPreferences = PreferencesPage(
	PreferencesGroup(
		SwitchRow().
			Title(gettext.Get("Start in Fullscreen")).
			Subtitle(gettext.Get("Open the player in fullscreen mode by default.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Experimental().BindStartInFullscreen(&sr.Object, "active")
				sr.SetVisible(preference.Experimental().EnableWindowedPlayer())
				preference.Experimental().OnEnableWindowedPlayerChanged(func() {
					sr.SetVisible(preference.Experimental().EnableWindowedPlayer())
				})
			}),
	).Title(gettext.Get("Windowed Player")),
).Title(gettext.Get("Player")).IconName("play")
