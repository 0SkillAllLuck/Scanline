package pages

import (
	"context"
	"log/slog"

	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/cards"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var HomeRoute = router.NewRoute("home", home)

func home(appCtx *appctx.AppContext) *router.Response {
	mgr := appCtx.Manager

	body := VStack().Spacing(25).VMargin(20)

	for _, src := range mgr.EnabledSources() {
		hubList, err := src.HomeHubs(context.Background())
		if err != nil {
			slog.Error("failed to fetch home hubs", "source", src.Name(), "error", err)
			continue
		}

		serverID := src.ID()
		coverURL := func(thumb string) string {
			return src.PhotoTranscodeURL(thumb, 240, 360)
		}

		for i := range hubList {
			hub := &hubList[i]
			list := lists.NewHorizontalList(hub.Title)
			hasItems := false

			// Continue Watching hub uses preview cards
			if hub.HubIdentifier == "home.continue" {
				for j := range hub.Metadata {
					meta := &hub.Metadata[j]
					artUrl := sources.ArtURL(meta)
					if artUrl != "" {
						switch meta.Type {
						case "movie":
							list.Append(cards.NewMoviePreviewCard(meta, src.PhotoTranscodeURL(artUrl, 480, 270), serverID))
							hasItems = true
						case "episode":
							list.Append(cards.NewEpisodePreviewCard(meta, src.PhotoTranscodeURL(artUrl, 480, 270), serverID))
							hasItems = true
						}
					}
				}
			} else {
				// Regular poster cards for other hubs
				hasItems = lists.RenderHub(list, hub, coverURL, serverID)
			}

			if hasItems {
				body = body.Append(list.SetPageMargin(40))
			}
		}
	}

	return &router.Response{
		PageTitle: gettext.Get("Home"),
		View: ScrolledWindow().
			Child(body).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}
