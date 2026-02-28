package lists

import (
	"math"

	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"codeberg.org/dergs/tonearm/pkg/schwifty/state"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

type HorizontalList struct {
	schwifty.Box

	container        *gtk.Box
	marginState      *state.State[int]
	routeButtonState *state.State[any]
}

func (h *HorizontalList) Append(child schwifty.BaseWidgetable) *HorizontalList {
	h.container.Append(child.ToGTK())
	return h
}

func (h *HorizontalList) SetPageMargin(margin int) *HorizontalList {
	h.marginState.SetValue(margin)
	return h
}

func (h *HorizontalList) SetViewAllRoute(path string) *HorizontalList {
	h.routeButtonState.SetValue(Button().Child(
		Label(gettext.Get("View All")).WithCSSClass("caption-heading"),
	).
		MinHeight(10).
		MinWidth(10).
		HPadding(10).
		ConnectClicked(func(b gtk.Button) {
			router.Navigate(path)
		}))
	return h
}

func NewHorizontalList(title string) *HorizontalList {
	marginState := state.NewStateful[int](0)
	routeButtonState := state.NewStateful[any](nil)
	container := HStack().BindHMargin(marginState)()

	var hAdjust *gtk.Adjustment
	nextButton := Button().
		MinHeight(10).MinWidth(10).HPadding(10).
		Child(Image().FromIconName("right-symbolic").PixelSize(10)).
		TooltipText(gettext.Get("Scroll to the right")).
		ConnectClicked(func(b gtk.Button) {
			current := hAdjust.GetValue()
			current -= math.Mod(current, 192)
			hAdjust.SetStepIncrement(192)
			go hAdjust.SetValue(current + hAdjust.GetStepIncrement())
		})

	previousButton := Button().
		MinHeight(10).MinWidth(10).HPadding(10).
		Child(Image().FromIconName("left-symbolic").PixelSize(10)).
		TooltipText(gettext.Get("Scroll to the left")).
		ConnectClicked(func(b gtk.Button) {
			current := hAdjust.GetValue()
			if math.Mod(current, 192) > 0 {
				current += 192 - math.Mod(current, 192)
			}
			hAdjust.SetStepIncrement(192)
			hAdjust.SetValue(current - hAdjust.GetStepIncrement())
		})

	return &HorizontalList{
		Box: VStack(
			HStack(
				HStack(
					Label(title).
						WithCSSClass("title-2").
						VAlign(gtk.AlignCenterValue),
					Spacer().VExpand(false),
					HStack(
						CenterBox().BindCenterWidget(routeButtonState).HExpand(false).VExpand(false),
						previousButton,
						nextButton,
					).Spacing(10),
				).BindHMargin(marginState),
			).HMargin(10).MarginBottom(5),
			ScrolledWindow().
				Child(Widget(&container.Widget)).
				VAlign(gtk.AlignStartValue).
				Policy(gtk.PolicyExternalValue, gtk.PolicyNeverValue).
				PropagateNaturalWidth(true).
				PropagateNaturalWidth(true).
				ConnectConstruct(func(sw *gtk.ScrolledWindow) {
					hAdjust = sw.GetHadjustment()
				}),
		),
		routeButtonState: routeButtonState,
		marginState:      marginState,
		container:        container,
	}
}
