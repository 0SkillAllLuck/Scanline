package auth

import (
	"context"
	"sync"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/dialogs/linking"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/utils/notifications"
	"github.com/0skillallluck/scanline/app/secrets"
	"github.com/0skillallluck/scanline/provider/plex/auth"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

// PerformSignIn runs the full Plex sign-in flow in a goroutine:
// clientID generation, PIN linking dialog, parallel GetUser+DiscoverServers, AddPlexAccount.
// ctx is used for cancellable network operations.
// window is used for the linking dialog's "Open in browser" button.
// presentOn is the widget to present the dialog on (may differ from the window, e.g. a dialog).
// onLoading is called on the main thread after the linking dialog closes, before server discovery (may be nil).
// onError is called on the main thread if server discovery fails after loading started (may be nil).
// onSuccess is called on the main thread after the account is added (may be nil).
func PerformSignIn(ctx context.Context, window *gtk.Window, presentOn *gtk.Widget, mgr *sources.Manager, onLoading, onError, onSuccess func()) {
	go func() {
		var dialog *adw.AlertDialog
		clientID := secrets.GetOrCreateClientID()
		token, err := auth.StartPinLinking(clientID, func(pin *auth.Pin, authURL string, cancel context.CancelFunc) {
			schwifty.OnMainThreadOnce(func(u uintptr) {
				dialog = linking.NewLinking(window, authURL, cancel)()
				dialog.Present(presentOn)
			}, 0)
		})

		schwifty.OnMainThreadOncePure(func() {
			if dialog != nil {
				dialog.ForceClose()
			}
			if err == nil && onLoading != nil {
				onLoading()
			}
		})

		if err != nil {
			notifications.OnToast.Notify(gettext.Get("Sign in failed or aborted"))
			return
		}

		var (
			user      *auth.User
			resources []auth.Resource
			userErr   error
			discErr   error
		)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			user, userErr = auth.GetUser(ctx, token, clientID)
		}()
		go func() {
			defer wg.Done()
			resources, discErr = auth.DiscoverServers(ctx, token, clientID)
		}()
		wg.Wait()

		username := "Plex User"
		if userErr == nil && user.Username != "" {
			username = user.Username
		}

		if discErr != nil {
			notifications.OnToast.Notify(gettext.Get("Failed to discover servers"))
			if onError != nil {
				schwifty.OnMainThreadOncePure(onError)
			}
			return
		}

		mgr.AddPlexAccount(ctx, token, username, clientID, resources)
		notifications.OnToast.Notify(gettext.Get("Signed in to Plex"))

		if onSuccess != nil {
			schwifty.OnMainThreadOncePure(onSuccess)
		}
	}()
}
