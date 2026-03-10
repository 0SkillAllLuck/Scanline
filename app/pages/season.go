package pages

import (
	"context"
	"fmt"
	"log/slog"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/cards"
	"github.com/0skillallluck/scanline/app/components/player"
	"github.com/0skillallluck/scanline/app/components/widgets"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
)

var SeasonRoute = router.NewRoute("season/:server/:ratingKey", Season)

func Season(ctx context.Context, appCtx *appctx.AppContext, serverID, ratingKey string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("Season"), errSourceNotFound(serverID))
	}

	meta, err := src.GetMetadata(ctx, ratingKey)
	if err != nil {
		return router.FromError(gettext.Get("Season"), err)
	}

	episodes, err := src.GetChildren(ctx, ratingKey)
	if err != nil {
		slog.Warn("failed to fetch episodes", "ratingKey", ratingKey, "error", err)
	}

	body := VStack().Spacing(25).VMargin(20).HMargin(40)

	// Find the next episode to play
	nextEpisode := findNextSeasonEpisode(episodes)

	// Hero section
	var buildButtonRow func() schwifty.Box
	if nextEpisode != nil && len(nextEpisode.Media) > 0 && len(nextEpisode.Media[0].Part) > 0 {
		ep := nextEpisode
		playLabel := gettext.Get("Play")
		if meta.ViewedLeafCount > 0 && meta.ViewedLeafCount < meta.LeafCount {
			if ep.ViewOffset > 0 {
				playLabel = fmt.Sprintf("%s %s", gettext.Get("Continue"), widgets.FormatTimestamp(ep.ViewOffset))
			} else {
				playLabel = fmt.Sprintf("%s %s", gettext.Get("Continue"), widgets.FormatEpisodeLabel(ep.ParentIndex, ep.Index))
			}
		}
		buildButtonRow = func() schwifty.Box {
			return HStack().Spacing(10).
				Append(
					Button().
						Child(
							HStack(
								Image().FromIconName("media-playback-start-symbolic"),
								Label(playLabel),
							).Spacing(6),
						).
						TooltipText(gettext.Get("Play this episode")).
						WithCSSClass("suggested-action").
						WithCSSClass("pill").
						ConnectClicked(func(b gtk.Button) {
							player.NewPlayer(player.PlayerParams{
								Ctx:        ctx,
								Title:      ep.Title,
								PartKey:    ep.Media[0].Part[0].Key,
								Window:     appCtx.Window,
								RatingKey:  ep.RatingKey,
								Media:      ep.Media,
								Source:     src,
								ViewOffset: ep.ViewOffset,
							})
						}),
				)
		}
	}

	heroContent := widgets.HeroContent(widgets.HeroContentParams{
		Title:            meta.ParentTitle,
		TitleActionName:  "win.route.show",
		TitleActionValue: serverID + "/" + meta.ParentRatingKey,
		Subtitle:         meta.Title,
		SubtitleClass:    "title-2 dimmed",
		Badges:           []string{fmt.Sprint(meta.Year)},
		BuildButtonRow:   buildButtonRow,
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

// findNextSeasonEpisode finds the next episode to play within a single season.
// It returns the first in-progress or unwatched episode.
// If all episodes are watched, it returns the first episode.
func findNextSeasonEpisode(episodes []sources.Metadata) *sources.Metadata {
	if len(episodes) == 0 {
		return nil
	}

	for i := range episodes {
		ep := &episodes[i]
		if ep.ViewOffset > 0 || ep.ViewCount == 0 {
			return ep
		}
	}

	return &episodes[0]
}
