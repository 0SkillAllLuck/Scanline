package secretservice

import (
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/secrets"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

// PresentSecretServiceErrorDialog creates and presents a dialog to display secret service errors.
// Does nothing if the warning should be hidden based on user settings.
func PresentSecretServiceErrorDialog(err *secrets.ServiceError, widget *gtk.Widget) {
	if preference.General().ShouldHideSecretServiceWarning() {
		return
	}

	// ConnectResponse is broken with puregotk, so we have to manually hack our way
	AlertDialog(err.Title, err.Body).
		WithCSSClass("no-response").
		ConnectConstruct(func(ad *adw.AlertDialog) {
			checkbox := gtk.NewCheckButtonWithLabel(gettext.Get("Don't show again"))
			checkbox.SetHalign(gtk.AlignBaselineCenterValue)
			checkbox.AddCssClass("space-2")

			ad.SetExtraChild(
				VStack(
					checkbox,
					Button().Label(gettext.Get("Continue")).WithCSSClass("destructive-action").VPadding(10).ConnectClicked(func(b gtk.Button) {
						if checkbox.GetActive() {
							preference.General().SetHideSecretServiceWarning(true)
						}
						ad.Close()
					}),
				).Spacing(12).ToGTK(),
			)
			ad.Present(widget)
		})()
}
