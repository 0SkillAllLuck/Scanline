package pages

import (
	"context"
	"fmt"
	"log/slog"

	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/cards"
	"github.com/0skillallluck/scanline/app/components/widgets"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var SeasonRoute = router.NewRoute("season/:server/:ratingKey", Season)

func Season(appCtx *appctx.AppContext, serverID, ratingKey string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("Season"), errSourceNotFound(serverID))
	}

	ctx := context.Background()

	meta, err := src.GetMetadata(ctx, ratingKey)
	if err != nil {
		return router.FromError(gettext.Get("Season"), err)
	}

	episodes, err := src.GetChildren(ctx, ratingKey)
	if err != nil {
		slog.Warn("failed to fetch episodes", "ratingKey", ratingKey, "error", err)
	}

	body := VStack().Spacing(25).VMargin(20).HMargin(40)

	// Hero section
	heroContent := widgets.HeroContent(widgets.HeroContentParams{
		Title:         meta.ParentTitle,
		Subtitle:      meta.Title,
		SubtitleClass: "title-2 dimmed",
		Badges:        []string{fmt.Sprint(meta.Year)},
	})

	hero := widgets.HeroSection(
		widgets.HeroPosterParams{
			ImageURL: src.PhotoTranscodeURL(meta.ParentThumb, 240, 360),
			Width:    240,
			Height:   360,
		},
		heroContent,
	)

	body = body.Append(hero)

	// Episodes section
	if len(episodes) > 0 {
		body = body.Append(
			Label(gettext.GetN("%d Episode", "%d Episodes", len(episodes), len(episodes))).
				WithCSSClass("title-4").
				HAlign(gtk.AlignStartValue),
		)

		episodeGrid := WrapBox()
		for i := range episodes {
			ep := &episodes[i]
			episodeGrid = episodeGrid.Append(cards.NewSeasonEpisode(ep, src.PhotoTranscodeURL(ep.Thumb, 320, 180), serverID))
		}
		body = body.Append(episodeGrid)
	}

	pageTitle := meta.Title
	if meta.ParentTitle != "" {
		pageTitle = meta.ParentTitle + " - " + meta.Title
	}

	return &router.Response{
		PageTitle: pageTitle,
		View: ScrolledWindow().
			Child(Clamp().MaximumSize(1200).Child(body)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}
