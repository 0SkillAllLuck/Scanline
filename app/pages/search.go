package pages

import (
	"context"
	"log/slog"
	"time"

	"codeberg.org/dergs/tonearm/pkg/schwifty/state"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/pages/search"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
)

var sortLabels = []string{
	gettext.Get("Title (A-Z)"),
	gettext.Get("Title (Z-A)"),
	gettext.Get("Year (Newest)"),
	gettext.Get("Year (Oldest)"),
	gettext.Get("Recently Added"),
}

type cachedSource struct {
	hubs     []sources.Hub
	coverURL func(string) string
	serverID string
}

var SearchRoute = router.NewRoute("search", func(ctx context.Context, appCtx *appctx.AppContext) *router.Response {
	scrollChildState := state.NewStateful[any](search.PromptView())
	searchState := state.NewStateful(false)
	var cached []cachedSource

	sortDD := gtk.NewDropDownFromStrings(sortLabels)
	sortDD.SetSelected(0)

	renderResults := func() {
		if len(cached) == 0 {
			return
		}
		sortOption := int(sortDD.GetSelected())
		body := VStack().Spacing(25).VMargin(20)
		for _, cs := range cached {
			search.SortHubs(cs.hubs, sortOption)
			results := search.Results(cs.hubs, cs.coverURL, cs.serverID)
			body = body.Append(results)
		}
		scrollChildState.SetValue(body.VAlign(gtk.AlignStartValue))
	}

	sortDD.ConnectSignal("notify::selected", new(func() {
		renderResults()
	}))

	searchHandler := func(searchBar gtk.SearchEntry) {
		query := searchBar.GetText()
		if query == "" {
			cached = nil
			scrollChildState.SetValue(search.PromptView())
			return
		}
		scrollChildState.SetValue(search.LoadingView())
		go func() {
			mgr := appCtx.Manager
			var newCached []cachedSource

			for _, src := range mgr.EnabledSources() {
				hubs, err := src.Search(ctx, query, 50)
				if err != nil {
					slog.Error("search failed", "source", src.Name(), "error", err)
					continue
				}
				srcID := src.ID()
				newCached = append(newCached, cachedSource{
					hubs: hubs,
					coverURL: func(thumb string) string {
						return src.PhotoTranscodeURL(thumb, 240, 360)
					},
					serverID: srcID,
				})
			}

			if len(newCached) == 0 {
				scrollChildState.SetValue(search.NoResultsView())
				return
			}
			cached = newCached
			renderResults()
		}()
	}

	return &router.Response{
		PageTitle: gettext.Get("Search"),
		Toolbar: HStack(
			SearchEntry().
				HExpand(true).
				PlaceholderText(gettext.Get("E.g. The Matrix")).
				SearchDelay(1000).
				ConnectActivate(func(se gtk.SearchEntry) {
					searchState.SetValue(true)
					time.AfterFunc(time.Second, func() {
						searchState.SetValue(false)
					})
					searchHandler(se)
				}).
				ConnectMap(func(w gtk.Widget) {
					w.GrabFocus()
				}).
				ConnectSearchChanged(func(se gtk.SearchEntry) {
					if searchState.Value() && se.GetText() != "" {
						return
					}
					searchHandler(se)
				}).
				AddController(escKeyController()),
			Widget(&sortDD.Widget),
		).Spacing(8).MarginEnd(40),
		View: ScrolledWindow().
			BindChild(scrollChildState).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
})

func escKeyController() *gtk.EventController {
	keyCtrl := gtk.NewEventControllerKey()
	keyCtrl.ConnectKeyPressed(new(func(ctrl gtk.EventControllerKey, keyval uint32, keycode uint32, state gdk.ModifierType) bool {
		if keyval == uint32(gdk.KEY_Escape) {
			router.Back()
			return true
		}
		return false
	}))
	return &keyCtrl.EventController
}
