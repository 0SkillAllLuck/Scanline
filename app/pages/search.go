package pages

import (
	"context"
	"log/slog"
	"time"

	"codeberg.org/dergs/tonearm/pkg/schwifty/state"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/pages/search"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var SearchRoute = router.NewRoute("search", func(appCtx *appctx.AppContext) *router.Response {
	scrollChildState := state.NewStateful[any](search.PromptView())
	searchState := state.NewStateful(false)
	searchHandler := onSearch(appCtx, scrollChildState)

	return &router.Response{
		PageTitle: gettext.Get("Search"),
		Toolbar: SearchEntry().
			HExpand(true).
			MarginEnd(40).
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
			}),
		View: ScrolledWindow().
			BindChild(scrollChildState).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
})

func onSearch(appCtx *appctx.AppContext, scrollChildState *state.State[any]) func(gtk.SearchEntry) {
	return func(searchBar gtk.SearchEntry) {
		query := searchBar.GetText()
		if query == "" {
			scrollChildState.SetValue(search.PromptView())
			return
		}
		scrollChildState.SetValue(search.LoadingView())
		go func() {
			mgr := appCtx.Manager

			body := VStack().Spacing(25).VMargin(20)
			hasResults := false

			for _, src := range mgr.EnabledSources() {
				hubs, err := src.Search(context.Background(), query, 50)
				if err != nil {
					slog.Error("search failed", "source", src.Name(), "error", err)
					continue
				}

				results := search.Results(hubs, func(thumb string) string {
					return src.PhotoTranscodeURL(thumb, 240, 360)
				}, src.ID())
				body = body.Append(results)
				hasResults = true
			}

			if !hasResults {
				scrollChildState.SetValue(search.NoResultsView())
				return
			}
			scrollChildState.SetValue(body.VAlign(gtk.AlignStartValue))
		}()
	}
}
