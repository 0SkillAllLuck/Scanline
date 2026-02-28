package preferences

import (
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var generalPreferences = PreferencesPage(
	PreferencesGroup(
		SpinRow(
			gtk.NewAdjustment(0, 1, 100, 1, 1, 1),
			1,
			0,
		).Title(gettext.Get("History Length")).
			Subtitle(gettext.Get("Maximum history length before dropping old entries.")).
			ConnectConstruct(func(sr *adw.SpinRow) {
				preference.Performance().BindMaxRouterHistorySize(&sr.Object, "value")
			}),
	).
		Title(gettext.Get("Navigation Behaviour")).
		Description(gettext.Get("Configure the behaviour of Scanline when navigating between pages.")),
).Title(gettext.Get("General")).IconName("settings-symbolic")
