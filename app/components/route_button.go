package components

import (
	"strings"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/internal/signals"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

type RouteButton struct {
	*gtk.Button
	label        *gtk.Label
	icon         *gtk.Image
	subscription *signals.Subscription
}

func (r *RouteButton) setActive(active bool) {
	if active {
		r.AddCssClass("active")
	} else {
		r.RemoveCssClass("active")
	}
}

func (r *RouteButton) Title(title string) *RouteButton {
	r.label.SetText(title)
	r.label.SetVisible(title != "")
	if title == "" {
		r.RemoveCssClass("title")
	} else {
		r.AddCssClass("title")
	}
	return r
}

func (r *RouteButton) Icon(iconName string) *RouteButton {
	r.icon.SetFromIconName(iconName)
	return r
}

func (r *RouteButton) TooltipText(tooltip string) *RouteButton {
	r.Button.SetTooltipText(tooltip)
	return r
}

func NewRouteButton(path string) *RouteButton {
	routeButton := &RouteButton{
		icon:  Image().FromIconName("image-missing-symbolic")(),
		label: Label("").PaddingStart(7).PaddingEnd(7).Visible(false)(),
	}
	routeButton.Button = Button().
		PaddingStart(0).
		PaddingEnd(0).
		MinHeight(24).
		ConnectClicked(func(b gtk.Button) {
			router.NavigateClearing(path)
		}).
		Child(
			HStack(
				Clamp().MaximumSize(16).Child(&routeButton.icon.Widget),
				routeButton.label,
			).
				Spacing(7).
				HMargin(9).
				VMargin(2),
		).
		WithCSSClass("flat").
		ConnectDestroy(func(w gtk.Widget) {
			if routeButton.subscription != nil {
				router.NavigationStarted.Unsubscribe(routeButton.subscription)
				routeButton.subscription = nil
			}
		})()

	routeButton.subscription = router.NavigationStarted.On(func(newPath string) bool {
		schwifty.OnMainThreadOnce(func(u uintptr) {
			if strings.HasPrefix(newPath, path) {
				routeButton.RemoveCssClass("flat")
				routeButton.AddCssClass("raised")
			} else {
				routeButton.RemoveCssClass("raised")
				routeButton.AddCssClass("flat")
			}
		}, 0)
		return signals.Continue
	})

	return routeButton
}
