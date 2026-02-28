package sources

import (
	"context"
	"fmt"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	appauth "github.com/0skillallluck/scanline/app/auth"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

func NewSourceSelection(window *gtk.Window, mgr *sources.Manager, onDone func()) *adw.Dialog {
	ctx, cancel := context.WithCancel(context.Background())

	dialog := adw.NewDialog()
	dialog.SetTitle(gettext.Get("Select Sources"))
	dialog.SetContentWidth(600)
	dialog.SetContentHeight(700)

	toolbarView := adw.NewToolbarView()

	headerBar := adw.NewHeaderBar()
	addButton := gtk.NewButtonFromIconName("list-add-symbolic")
	addButton.SetTooltipText(gettext.Get("Add Account"))
	headerBar.PackStart(&addButton.Widget)
	toolbarView.AddTopBar(&headerBar.Widget)

	dialog.SetChild(&toolbarView.Widget)

	dialog.ConnectCloseAttempt(new(func(d adw.Dialog) {
		cancel()
		onDone()
	}))

	// refreshContent rebuilds the dialog content from current manager state.
	// We use a variable so it can be called recursively from sign-out handlers.
	var refreshContent func()
	refreshContent = func() {
		accounts := mgr.Accounts()

		if len(accounts) == 0 {
			emptyContent := VStack(
				Label(gettext.Get("No accounts configured.")).WithCSSClass("dim-label"),
				Button().
					Label(gettext.Get("Add Account")).
					WithCSSClass("pill").
					WithCSSClass("suggested-action").
					HAlign(gtk.AlignCenterValue).
					ConnectClicked(func(b gtk.Button) {
						addButton.Activate()
					}),
			).Spacing(12).VAlign(gtk.AlignCenterValue).VExpand(true).ToGTK()
			toolbarView.SetContent(emptyContent)
			return
		}

		var groupWidgets []any
		for _, acct := range accounts {
			acctID := acct.ID
			var rows []any

			for _, srv := range acct.Servers {
				srvID := srv.ID
				srvEnabled := srv.Enabled
				acctIDCopy := acctID

				rows = append(rows,
					SwitchRow().
						Title(srv.Name).
						Subtitle(serverStatusText(srv)).
						ConnectConstruct(func(row *adw.SwitchRow) {
							row.SetActive(srvEnabled)
							cb := func() {
								mgr.SetServerEnabled(acctIDCopy, srvID, row.GetActive())
							}
							row.ConnectSignal("notify::active", &cb)
						}),
				)
			}

			// Sign Out button
			signOutAcctID := acctID
			rows = append(rows,
				ButtonRow().
					Title(gettext.Get("Sign Out")).
					StartIconName("system-log-out-symbolic").
					ConnectConstruct(func(br *adw.ButtonRow) {
						cb := func(adw.ButtonRow) {
							mgr.RemoveAccount(signOutAcctID)
							if mgr.HasAccounts() {
								schwifty.OnMainThreadOncePure(refreshContent)
							} else {
								schwifty.OnMainThreadOncePure(onDone)
							}
						}
						br.ConnectActivated(&cb)
					}),
			)

			groupTitle := acct.Username + " (Plex)"
			groupWidgets = append(groupWidgets,
				PreferencesGroup(rows...).Title(groupTitle),
			)
		}

		content := VStack(groupWidgets...).Spacing(12).HMargin(12).VMargin(12).ToGTK()
		scrolled := ScrolledWindow().
			Child(content).
			PropagateNaturalHeight(true)
		toolbarView.SetContent(scrolled.ToGTK())
	}

	showLoading := func() {
		loadingContent := VStack(
			Spinner().SizeRequest(32, 32),
			Label(gettext.Get("Discovering servers...")).WithCSSClass("dim-label"),
		).Spacing(20).VAlign(gtk.AlignCenterValue).VExpand(true).ToGTK()
		toolbarView.SetContent(loadingContent)
	}

	// "+" button: add new Plex account
	addButton.ConnectClicked(new(func(b gtk.Button) {
		onSuccess := refreshContent
		if !mgr.HasAccounts() {
			onSuccess = onDone
		}
		appauth.PerformSignIn(ctx, window, &dialog.Widget, mgr, showLoading, refreshContent, onSuccess)
	}))

	// Initial load
	if mgr.HasAccounts() {
		refreshContent()
	} else {
		loadingContent := VStack(
			Spinner().SizeRequest(32, 32),
			Label(gettext.Get("Discovering servers...")).WithCSSClass("dim-label"),
		).Spacing(20).VAlign(gtk.AlignCenterValue).VExpand(true).ToGTK()
		toolbarView.SetContent(loadingContent)

		go func() {
			mgr.RefreshServers(ctx)
			schwifty.OnMainThreadOncePure(refreshContent)
		}()
	}

	return dialog
}

func serverStatusText(srv *sources.Server) string {
	if srv.URL == "" {
		return gettext.Get("Not connected")
	}
	return fmt.Sprintf("%s", srv.URL)
}
