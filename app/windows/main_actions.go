package windows

import (
	"unsafe"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/dialogs/about"
	"github.com/0skillallluck/scanline/app/dialogs/preferences"
	"github.com/0skillallluck/scanline/app/dialogs/shortcuts"
	"github.com/0skillallluck/scanline/app/dialogs/sources"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gio"
	"github.com/jwijenbergh/puregotk/v4/glib"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

// installAppActions installs actions that live for the entire app lifecycle:
// about, preferences, shortcuts, quit.
func (w *Window) installAppActions() {
	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		dialog := about.NewAboutDialog()
		dialog.Present(&w.Widget)
		dialog.Unref()
	}))
	w.GetApplication().Application.AddAction(aboutAction)

	preferencesAction := gio.NewSimpleAction("preferences", nil)
	preferencesAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		preferences.NewPreferencesDialog().Present(w)
	}))
	w.GetApplication().Application.AddAction(preferencesAction)
	w.GetApplication().SetAccelsForAction("app.preferences", []string{"<Control>comma"})

	shortcutsAction := gio.NewSimpleAction("shortcuts", nil)
	shortcutsAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		shortcuts.NewShortcutsDialog().Present(w)
	}))
	w.GetApplication().Application.AddAction(shortcutsAction)
	w.GetApplication().SetAccelsForAction("app.shortcuts", []string{"<Control>question"})

	quitAction := gio.NewSimpleAction("quit", nil)
	quitAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		w.GetApplication().Quit()
	}))
	w.GetApplication().Application.AddAction(quitAction)
	w.GetApplication().SetAccelsForAction("app.quit", []string{"<Ctrl>q"})

	closeAction := gio.NewSimpleAction("close", nil)
	closeAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		w.Close()
	}))
	w.AddAction(closeAction)
	w.GetApplication().SetAccelsForAction("win.close", []string{"<Ctrl>w"})
}

// installWindowActions installs actions that only make sense when main content is shown:
// sign-in, select-sources, navigate-back, search, routes.
func (w *Window) installWindowActions() {
	selectSourcesAction := gio.NewSimpleAction("select-sources", nil)
	selectSourcesAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		w.presentSourceSelection()
	}))
	w.AddAction(selectSourcesAction)

	navigateBackAction := gio.NewSimpleAction("navigate-back", nil)
	navigateBackAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		router.Back()
	}))
	w.AddAction(navigateBackAction)
	w.GetApplication().SetAccelsForAction("win.navigate-back", []string{"<Alt>Left"})

	searchAction := gio.NewSimpleAction("search", nil)
	searchAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		router.Navigate("search")
	}))
	w.AddAction(searchAction)
	w.GetApplication().SetAccelsForAction("win.search", []string{"<Ctrl>f"})

	routeMovieAction := gio.NewSimpleAction("route.movie", glib.NewVariantType("s"))
	routeMovieAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		variant := (*glib.Variant)(unsafe.Pointer(parameter))
		router.Navigate("movie/" + variant.GetString(nil))
	}))
	w.AddAction(routeMovieAction)

	routeShowAction := gio.NewSimpleAction("route.show", glib.NewVariantType("s"))
	routeShowAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		variant := (*glib.Variant)(unsafe.Pointer(parameter))
		router.Navigate("show/" + variant.GetString(nil))
	}))
	w.AddAction(routeShowAction)

	routeSeasonAction := gio.NewSimpleAction("route.season", glib.NewVariantType("s"))
	routeSeasonAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		variant := (*glib.Variant)(unsafe.Pointer(parameter))
		router.Navigate("season/" + variant.GetString(nil))
	}))
	w.AddAction(routeSeasonAction)

	routeEpisodeAction := gio.NewSimpleAction("route.episode", glib.NewVariantType("s"))
	routeEpisodeAction.ConnectActivate(new(func(action gio.SimpleAction, parameter uintptr) {
		variant := (*glib.Variant)(unsafe.Pointer(parameter))
		router.Navigate("episode/" + variant.GetString(nil))
	}))
	w.AddAction(routeEpisodeAction)
}

func (w *Window) presentSourceSelection() {
	mgr := w.appCtx.Manager
	var dialog *adw.Dialog
	schwifty.OnMainThreadOnce(func(u uintptr) {
		dialog = sources.NewSourceSelection(w.appCtx.Ctx, &w.Window, mgr, func() {
			dialog.ForceClose()
			router.Refresh()
		})
		dialog.Present(&w.Widget)
	}, 0)
}

const (
	MouseButtonBack    = 8
	MouseButtonForward = 9
)

func (w *Window) installMouseClickHandler() {
	gestureController := gtk.NewGestureClick()
	gestureController.SetButton(0)
	gestureController.SetPropagationPhase(gtk.PhaseCaptureValue)
	gestureController.ConnectPressed(new(func(controller gtk.GestureClick, nPress int, x float64, y float64) {
		switch controller.GetCurrentButton() {
		case MouseButtonBack:
			w.ActivateAction("navigate-back", nil)
		}
	}))
	w.AddController(&gestureController.EventController)
}
