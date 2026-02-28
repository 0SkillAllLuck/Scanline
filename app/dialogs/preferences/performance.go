package preferences

import (
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/cacheutils"
	"github.com/jwijenbergh/puregotk/v4/adw"
)

var performancePreferences = PreferencesPage(
	PreferencesGroup(
		SwitchRow().
			Title(gettext.Get("Allow Preview Images")).
			Subtitle(gettext.Get("Allow Scanline to load images for the \"Continue Watching\" section.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Performance().BindAllowPreviewImages(&sr.Object, "active")
			}),
		SwitchRow().
			Title(gettext.Get("Allow Poster Images")).
			Subtitle(gettext.Get("Allow Scanline to load images poster images for libraries and details pages.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Performance().BindAllowPosterImages(&sr.Object, "active")
			}),
	).Title(gettext.Get("Images")),
	PreferencesGroup(
		SwitchRow().
			Title(gettext.Get("Cache Images")).
			Subtitle(gettext.Get("Cache images locally to improve performance and reduce network traffic.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Performance().BindCacheImages(&sr.Object, "active")
			}),
		SwitchRow().
			Title(gettext.Get("Cache Libraries")).
			Subtitle(gettext.Get("Cache library content locally to improve performance and reduce network traffic.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Performance().BindCacheLibraries(&sr.Object, "active")
			}),
		SwitchRow().
			Title(gettext.Get("Cache Metadata")).
			Subtitle(gettext.Get("Cache metadata locally to improve performance and reduce network traffic.")).
			ConnectConstruct(func(sr *adw.SwitchRow) {
				preference.Performance().BindCacheMetadata(&sr.Object, "active")
			}),
		ButtonRow().
			Title(gettext.Get("Clear Cache")).
			StartIconName("brush-symbolic").
			ConnectConstruct(func(br *adw.ButtonRow) {
				cb := func(adw.ButtonRow) {
					cacheutils.Clear()
				}
				br.ConnectActivated(&cb)
			}),
	).Title(gettext.Get("Caching")),
).Title(gettext.Get("Performance")).IconName("speedometer5-symbolic")
