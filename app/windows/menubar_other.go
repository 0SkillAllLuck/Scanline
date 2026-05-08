//go:build !darwin

package windows

import (
	adwbindings "codeberg.org/dergs/tonearm/pkg/schwifty/bindings/adw"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gio"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/internal/gettext"
)

func (w *Window) installNativeMenubar() {}

func (w *Window) buildMainMenu() *gio.Menu {
	mainMenu := gio.NewMenu()
	mainMenu.Append(gettext.Get("Select Sources"), "win.select-sources")
	mainMenu.Append(gettext.Get("Preferences"), "app.preferences")
	mainMenu.Append(gettext.Get("Keyboard Shortcuts"), "app.shortcuts")
	mainMenu.Append(gettext.Get("About Scanline"), "app.about")
	return mainMenu
}

func (w *Window) packMainMenuButton(b adwbindings.HeaderBar) adwbindings.HeaderBar {
	mainMenu := w.buildMainMenu()
	return b.PackEnd(
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
	)
}
