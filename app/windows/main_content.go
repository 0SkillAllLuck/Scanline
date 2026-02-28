package windows

import (
	"context"
	"log/slog"
	"strings"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"codeberg.org/dergs/tonearm/pkg/schwifty/state"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/components"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/internal/signals"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gio"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var decorationLayoutState = state.NewStateful("icon,appmenu:close")

func updateDecorationLayout() {
	settings := gtk.SettingsGetDefault()
	defer settings.Unref()

	configured := settings.GetPropertyGtkDecorationLayout()
	splits := strings.Split(configured, ":")
	left := splits[0]
	right := ""
	if len(splits) > 1 {
		right = splits[1]
	}

	if left == "appmenu" {
		decorationLayoutState.SetValue("icon," + left + ":" + right)
	} else if right == "appmenu" {
		decorationLayoutState.SetValue(left + ":" + right + ",icon")
	} else {
		decorationLayoutState.SetValue(configured)
	}
}

func (w *Window) buildMainMenu() *gio.Menu {
	mainMenu := gio.NewMenu()
	mainMenu.Append(gettext.Get("Select Sources"), "win.select-sources")
	mainMenu.Append(gettext.Get("Preferences"), "app.preferences")
	mainMenu.Append(gettext.Get("Keyboard Shortcuts"), "app.shortcuts")
	mainMenu.Append(gettext.Get("About Scanline"), "app.about")
	return mainMenu
}

func iconForSectionType(sectionType string) string {
	switch sectionType {
	case "movie":
		return "camera-video-symbolic"
	case "show":
		return "tv-symbolic"
	case "artist":
		return "audio-x-generic-symbolic"
	case "photo":
		return "image-x-generic-symbolic"
	default:
		return "folder-symbolic"
	}
}

func (w *Window) buildContentHeader() *gtk.Widget {
	mgr := w.appCtx.Manager

	homeButton := components.NewRouteButton("home")
	homeButton.Title(gettext.Get("Home"))
	homeButton.Icon("go-home-symbolic")
	homeButton.TooltipText(gettext.Get("Navigate to Home"))

	watchlistButton := components.NewRouteButton("watchlist")
	watchlistButton.Title(gettext.Get("Watchlist"))
	watchlistButton.Icon("starred-symbolic")
	watchlistButton.SetVisible(false)

	defaultToolbar := HStack(
		Widget(&homeButton.Widget),
		Widget(&watchlistButton.Widget),
	).Spacing(3)()

	// We never want to delete the default toolbar. NEVER.
	defaultToolbar.Ref()

	var libraryButtons []*components.RouteButton

	refreshLibraryButtons := func() {
		go func() {
			enabledSources := mgr.EnabledSources()
			if len(enabledSources) == 0 {
				schwifty.OnMainThreadOncePure(func() {
					for _, btn := range libraryButtons {
						defaultToolbar.Remove(&btn.Widget)
					}
					libraryButtons = nil
				})
				return
			}

			type sectionInfo struct {
				serverID string
				section  sources.LibrarySection
			}
			var allSections []sectionInfo

			for _, src := range enabledSources {
				sections, err := src.LibrarySections(context.Background())
				if err != nil {
					slog.Error("failed to fetch library sections", "source", src.Name(), "error", err)
					continue
				}
				for _, s := range sections {
					allSections = append(allSections, sectionInfo{serverID: src.ID(), section: s})
				}
			}

			// Check for duplicate section names across servers
			nameCounts := make(map[string]int)
			for _, si := range allSections {
				nameCounts[si.section.Title]++
			}

			schwifty.OnMainThreadOncePure(func() {
				for _, btn := range libraryButtons {
					defaultToolbar.Remove(&btn.Widget)
				}
				libraryButtons = nil

				for _, si := range allSections {
					title := si.section.Title
					if nameCounts[title] > 1 {
						// Find the source name for disambiguation
						if src := mgr.SourceForServer(si.serverID); src != nil {
							title += " (" + src.Name() + ")"
						}
					}
					btn := components.NewRouteButton("library/" + si.serverID + "/" + si.section.Key)
					btn.Title(title)
					btn.Icon(iconForSectionType(si.section.Type))
					defaultToolbar.Append(&btn.Widget)
					libraryButtons = append(libraryButtons, btn)
				}
			})
		}()
	}

	// Update visibility based on sources
	updateVisibility := func() {
		hasAccounts := mgr.HasAccounts()
		hasSources := len(mgr.EnabledSources()) > 0
		schwifty.OnMainThreadOncePure(func() {
			homeButton.SetVisible(hasAccounts)
			watchlistButton.SetVisible(hasSources && preference.Experimental().EnableWatchlist())
		})
	}

	mgr.SourcesChanged.On(func(_ struct{}) bool {
		updateVisibility()
		refreshLibraryButtons()
		return signals.Continue
	})

	preference.Experimental().OnEnableWatchlistChanged(func() {
		schwifty.OnMainThreadOncePure(func() {
			watchlistButton.SetVisible(
				len(mgr.EnabledSources()) > 0 && preference.Experimental().EnableWatchlist(),
			)
		})
	})

	mainMenu := w.buildMainMenu()

	gtkSettings := gtk.SettingsGetDefault()
	gtkSettings.ConnectSignal("notify::gtk-decoration-layout", new(func() {
		updateDecorationLayout()
	}))
	updateDecorationLayout()

	hasSources := len(mgr.EnabledSources()) > 0

	headerbar := HeaderBar().
		BindDecorationLayout(decorationLayoutState).
		CenteringPolicy(adw.CenteringPolicyStrictValue).
		PackStart(
			Button().
				IconName("loupe-symbolic").
				ActionName("win.search").
				TooltipText(gettext.Get("Search")).
				Visible(hasSources).
				ConnectConstruct(func(b *gtk.Button) {
					var sub *signals.Subscription
					sub = mgr.SourcesChanged.On(func(_ struct{}) bool {
						schwifty.OnMainThreadOncePure(func() {
							b.SetVisible(len(mgr.EnabledSources()) > 0)
						})
						return signals.Continue
					})
					b.ConnectDestroy(new(func(w gtk.Widget) {
						if sub != nil {
							mgr.SourcesChanged.Unsubscribe(sub)
							sub = nil
						}
					}))
				}),
			Button().
				IconName("left-symbolic").
				ActionName("win.navigate-back").
				Visible(false).
				TooltipText(gettext.Get("Navigate Back")).
				ConnectConstruct(func(b *gtk.Button) {
					var sub *signals.Subscription
					sub = router.HistoryUpdated.On(func(history *router.History) bool {
						schwifty.OnMainThreadOncePure(func() {
							b.SetVisible(len(history.Entries) > 0)
						})
						return signals.Continue
					})
					b.ConnectDestroy(new(func(w gtk.Widget) {
						if sub != nil {
							router.HistoryUpdated.Unsubscribe(sub)
							sub = nil
						}
					}))
				}),
		).
		TitleWidget(defaultToolbar).
		PackEnd(
			MenuButton().
				IconName("open-menu-symbolic").
				MenuModel(&mainMenu.MenuModel).
				TooltipText(gettext.Get("Main Menu")).ConnectConstruct(func(mb *gtk.MenuButton) {
				menuAction := gio.NewSimpleAction("main-menu", nil)
				menuAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
					mb.Popup()
				}))
				w.AddAction(menuAction)
				w.GetApplication().SetAccelsForAction("win.main-menu", []string{"F10"})
			}),
		).
		ConnectDestroy(func(w gtk.Widget) {
			gtkSettings.Unref()
		})()

	var navStartedSub, navCompletedSub *signals.Subscription
	navStartedSub = router.NavigationStarted.On(func(path string) bool {
		schwifty.OnMainThreadOnce(func(u uintptr) {
			headerbar.SetTitleWidget(&defaultToolbar.Widget)
		}, 0)
		return signals.Continue
	})

	navCompletedSub = router.NavigationCompleted.On(func(entry router.HistoryEntry) bool {
		schwifty.OnMainThreadOncePure(func() {
			if entry.Toolbar != nil {
				headerbar.SetTitleWidget(entry.Toolbar)
			} else {
				headerbar.SetTitleWidget(&defaultToolbar.Widget)
			}
		})
		return signals.Continue
	})

	headerbar.ConnectDestroy(new(func(w gtk.Widget) {
		if navStartedSub != nil {
			router.NavigationStarted.Unsubscribe(navStartedSub)
			navStartedSub = nil
		}
		if navCompletedSub != nil {
			router.NavigationCompleted.Unsubscribe(navCompletedSub)
			navCompletedSub = nil
		}
	}))

	// Initial load of library buttons
	updateVisibility()
	if hasSources {
		refreshLibraryButtons()
	}

	return &headerbar.Widget
}
