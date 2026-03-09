package pages

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/cards"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/components/player"
	"github.com/0skillallluck/scanline/app/components/widgets"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
)

var ShowRoute = router.NewRoute("show/:server/:ratingKey", Show)

func Show(appCtx *appctx.AppContext, serverID, ratingKey string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("TV Show"), errSourceNotFound(serverID))
	}

	ctx := appCtx.Ctx

	meta, err := src.GetMetadata(ctx, ratingKey)
	if err != nil {
		return router.FromError(gettext.Get("TV Show"), err)
	}

	seasons, err := src.GetChildren(ctx, ratingKey)
	if err != nil {
		slog.Warn("failed to fetch seasons", "ratingKey", ratingKey, "error", err)
	}
	relatedHubs, err := src.RelatedHubs(ctx, ratingKey)
	if err != nil {
		slog.Debug("failed to fetch related hubs", "ratingKey", ratingKey, "error", err)
	}

	// Find the next episode to play
	nextEpisode := findNextEpisode(ctx, src, seasons)

	body := VStack().Spacing(25).VMargin(20).HMargin(40)

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
								Ctx:        appCtx.Ctx,
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
		Title:          meta.Title,
		Subtitle:       meta.Tagline,
		Badges:         []string{fmt.Sprint(meta.Year), widgets.FormatSeasonCount(meta.ChildCount), meta.ContentRating},
		Ratings:        meta.Ratings,
		UserRating:     meta.UserRating,
		BuildButtonRow: buildButtonRow,
		Summary:        meta.Summary,
	})

	hero := widgets.HeroSection(
		widgets.HeroPosterParams{
			ImageURL: src.PhotoTranscodeURL(meta.Thumb, 240, 360),
			Width:    240,
			Height:   360,
		},
		heroContent,
	)

	body = body.Append(hero)

	// Seasons section
	if len(seasons) > 0 {
		seasonList := lists.NewHorizontalList(gettext.Get("Seasons"))
		for i := range seasons {
			s := &seasons[i]
			seasonList.Append(cards.NewSeasonPoster(s, src.PhotoTranscodeURL(s.Thumb, 240, 360), serverID))
		}
		body = body.Append(seasonList.SetPageMargin(0))
	}

	// Cast section
	if len(meta.Role) > 0 {
		castList := lists.NewHorizontalList(gettext.Get("Cast"))
		for _, role := range meta.Role {
			castList.Append(cards.NewCastMember(role.Tag, role.Role, src.PhotoTranscodeURL(role.Thumb, 140, 140), serverID, strconv.Itoa(role.ID)))
		}
		body = body.Append(castList.SetPageMargin(0))
	}

	// Related section
	coverURL := func(thumb string) string {
		return src.PhotoTranscodeURL(thumb, 240, 360)
	}
	for i := range relatedHubs {
		hub := &relatedHubs[i]
		list := lists.NewHorizontalList(hub.Title)
		if lists.RenderHub(list, hub, coverURL, serverID) {
			body = body.Append(list.SetPageMargin(0))
		}
	}

	return &router.Response{
		PageTitle: meta.Title,
		View: ScrolledWindow().
			Child(Clamp().MaximumSize(1200).Child(body)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}

// findNextEpisode finds the next episode to play for a show.
// It returns the first in-progress or unwatched episode, skipping specials (index 0).
// If all episodes are watched, it returns the first episode of the first non-specials season.
func findNextEpisode(ctx context.Context, src sources.Source, seasons []sources.Metadata) *sources.Metadata {
	if len(seasons) == 0 {
		return nil
	}

	var fallback *sources.Metadata

	for i := range seasons {
		season := &seasons[i]

		// Skip specials season
		if season.Index == 0 {
			continue
		}

		// Skip fully watched seasons, but not before recording a fallback
		if season.ViewedLeafCount >= season.LeafCount {
			if fallback == nil {
				episodes, err := src.GetChildren(ctx, season.RatingKey)
				if err == nil && len(episodes) > 0 {
					fallback = &episodes[0]
				}
			}
			continue
		}

		// This season has unwatched episodes - fetch them
		episodes, err := src.GetChildren(ctx, season.RatingKey)
		if err != nil {
			slog.Warn("failed to fetch episodes for season", "ratingKey", season.RatingKey, "error", err)
			continue
		}
		if len(episodes) == 0 {
			continue
		}

		if fallback == nil {
			fallback = &episodes[0]
		}

		for j := range episodes {
			ep := &episodes[j]
			if ep.ViewOffset > 0 || ep.ViewCount == 0 {
				return ep
			}
		}
	}

	return fallback
}
