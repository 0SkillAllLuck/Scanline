package windows

import (
	"sync"
	"time"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/dialogs/about"
	"github.com/0skillallluck/scanline/app/dialogs/sources"
	"github.com/0skillallluck/scanline/internal/g"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/utils/notifications"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/internal/signals"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gobject"
	"github.com/jwijenbergh/puregotk/v4/gtk"

	_ "github.com/0skillallluck/scanline/app/pages"
	_ "github.com/0skillallluck/scanline/app/pages/search"
)

type Window struct {
	*adw.ApplicationWindow
	appCtx *appctx.AppContext
}

var loadingView = g.Lazy(func() *gtk.Widget {
	widget := Clamp().MaximumSize(50).Child(Spinner()).ToGTK()
	widget.Ref()
	return widget
})

func NewWindow(app *adw.Application, appCtx *appctx.AppContext) *Window {
	window := &Window{
		ApplicationWindow: adw.NewApplicationWindow(&app.Application),
		appCtx:            appCtx,
	}

	window.SetTitle("Scanline")
	window.SetIconName("logo-symbolic")
	window.SetDefaultSize(preference.General().GetWindowWidth(), preference.General().GetWindowHeight())
	// Debounce window size saves to avoid excessive GSettings writes during resize
	var sizeTimer *time.Timer
	var sizeMu sync.Mutex
	window.ConnectNotify(new(func(gobject.Object, uintptr) {
		h := window.GetHeight()
		w := window.GetWidth()
		if h <= 0 && w <= 0 {
			return
		}
		sizeMu.Lock()
		if sizeTimer != nil {
			sizeTimer.Stop()
		}
		sizeTimer = time.AfterFunc(500*time.Millisecond, func() {
			if h > 0 {
				preference.General().SetWindowHeight(h)
			}
			if w > 0 {
				preference.General().SetWindowWidth(w)
			}
		})
		sizeMu.Unlock()
	}))

	if !about.IsStable() {
		window.AddCssClass("devel")
	}

	window.installAppActions()

	if appCtx.Manager.HasAccounts() {
		window.showMainContent()
	} else {
		window.showWelcomeContent()
	}

	return window
}

func (w *Window) showWelcomeContent() {
	toolbarView := adw.NewToolbarView()

	headerbar := HeaderBar().
		BindDecorationLayout(decorationLayoutState)()
	toolbarView.AddTopBar(&headerbar.Widget)

	mgr := w.appCtx.Manager
	welcomePage := StatusPage().
		IconName("avatar-default-symbolic").
		Title(gettext.Get("Welcome to Scanline")).
		Description(gettext.Get("Add your Plex media sources to get started.")).
		ConnectConstruct(func(sp *adw.StatusPage) {
			btn := Button().
				Label(gettext.Get("Add Sources")).
				WithCSSClass("pill").
				WithCSSClass("suggested-action").
				HAlign(gtk.AlignCenterValue).
				ConnectClicked(func(_ gtk.Button) {
					var dialog *adw.Dialog
					dialog = sources.NewSourceSelection(w.appCtx.Ctx, &w.Window, mgr, func() {
						dialog.ForceClose()
						if mgr.HasAccounts() {
							w.showMainContent()
						}
					})
					dialog.Present(&w.Widget)
				})()
			sp.SetChild(&btn.Widget)
		}).ToGTK()

	toolbarView.SetContent(welcomePage)

	w.setWindowContent(&toolbarView.Widget)
}

func (w *Window) showMainContent() {
	router.SetContext(w.appCtx, w.appCtx.Ctx)

	mgr := w.appCtx.Manager
	w.SetTitle(mgr.WindowTitle())

	w.installWindowActions()
	w.installMouseClickHandler()

	mainView, titleSub := w.buildMainView()
	w.setWindowContent(mainView)

	router.NavigateClearing("home")

	// Monitor for all accounts being removed → transition back to welcome
	accountSub := mgr.SourcesChanged.On(func(_ struct{}) bool {
		if !mgr.HasAccounts() {
			schwifty.OnMainThreadOncePure(func() {
				// Clean up subscriptions before transitioning
				if titleSub != nil {
					mgr.SourcesChanged.Unsubscribe(titleSub)
				}
				w.showWelcomeContent()
			})
			return signals.Unsubscribe
		}
		return signals.Continue
	})
	_ = accountSub
}

func (w *Window) setWindowContent(content *gtk.Widget) {
	toastLayout := adw.NewToastOverlay()
	toastLayout.SetChild(content)

	notifications.OnToast.On(func(title string) bool {
		schwifty.OnMainThreadOncePure(func() {
			toast := adw.NewToast(title)
			toast.SetTimeout(3)
			toastLayout.AddToast(toast)
		})
		return signals.Continue
	})

	w.SetContent(&toastLayout.Widget)
}

func (w *Window) buildMainView() (*gtk.Widget, *signals.Subscription) {
	mgr := w.appCtx.Manager

	sub := mgr.SourcesChanged.On(func(_ struct{}) bool {
		if !mgr.HasAccounts() {
			return signals.Unsubscribe
		}
		schwifty.OnMainThreadOncePure(func() {
			w.SetTitle(mgr.WindowTitle())
		})
		return signals.Continue
	})

	return w.buildContentLayout(), sub
}

func (w *Window) buildContentLayout() *gtk.Widget {
	toolbarView := adw.NewToolbarView()
	toolbarView.AddTopBar(w.buildContentHeader())

	var navStartedSub, navCompletedSub *signals.Subscription
	navStartedSub = router.NavigationStarted.On(func(path string) bool { //nolint:staticcheck // SA4006 - used in closure
		schwifty.OnMainThreadOnce(func(u uintptr) {
			toolbarView.SetContent(loadingView())
		}, 0)
		return signals.Continue
	})

	navCompletedSub = router.NavigationCompleted.On(func(entry router.HistoryEntry) bool { //nolint:staticcheck // SA4006 - used in closure
		schwifty.OnMainThreadOncePure(func() {
			toolbarView.SetContent(entry.View)
		})
		return signals.Continue
	})

	toolbarView.ConnectDestroy(new(func(w gtk.Widget) {
		if navStartedSub != nil {
			router.NavigationStarted.Unsubscribe(navStartedSub)
			navStartedSub = nil
		}
		if navCompletedSub != nil {
			router.NavigationCompleted.Unsubscribe(navCompletedSub)
			navCompletedSub = nil
		}
	}))

	return &toolbarView.Widget
}
