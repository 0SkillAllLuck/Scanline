package pages

import (
	"fmt"
	"log/slog"

	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/cards"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/components/widgets"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/jwijenbergh/puregotk/v4/gtk"
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

	body := VStack().Spacing(25).VMargin(20).HMargin(40)

	// Hero section
	heroContent := widgets.HeroContent(widgets.HeroContentParams{
		Title:      meta.Title,
		Subtitle:   meta.Tagline,
		Badges:     []string{fmt.Sprint(meta.Year), widgets.FormatSeasonCount(meta.ChildCount), meta.ContentRating},
		Ratings:    meta.Ratings,
		UserRating: meta.UserRating,
		Summary:    meta.Summary,
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
			castList.Append(cards.NewCastMember(role.Tag, role.Role, src.PhotoTranscodeURL(role.Thumb, 140, 140)))
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
