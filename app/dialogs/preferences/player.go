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
			}),
		SwitchRow().
			Title(gettext.Get("Auto Skip Intro")).
			Subtitle(gettext.Get("Automatically skip the intro when it begins playing.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Experimental().BindAutoSkipIntro(&sr.Object, "active")
			}),
		SwitchRow().
			Title(gettext.Get("Auto Skip Credits")).
			Subtitle(gettext.Get("Automatically skip the credits when they begin playing.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Experimental().BindAutoSkipCredits(&sr.Object, "active")
			}),
	).Title(gettext.Get("Player")),
).Title(gettext.Get("Player")).IconName("play")
