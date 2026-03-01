package pages

import (
	"fmt"
	"log/slog"
	"strings"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/cards"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/components/player"
	"github.com/0skillallluck/scanline/app/components/widgets"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/utils/notifications"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var MovieRoute = router.NewRoute("movie/:server/:ratingKey", Movie)

func Movie(appCtx *appctx.AppContext, serverID, ratingKey string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("Movie"), errSourceNotFound(serverID))
	}

	ctx := appCtx.Ctx

	meta, err := src.GetMetadata(ctx, ratingKey)
	if err != nil {
		return router.FromError(gettext.Get("Movie"), err)
	}

	relatedHubs, err := src.RelatedHubs(ctx, ratingKey)
	if err != nil {
		slog.Debug("failed to fetch related hubs", "ratingKey", ratingKey, "error", err)
	}

	body := VStack().Spacing(25).MarginTop(40).MarginBottom(20).HMargin(40)

	// Hero section
	var directorSubtitle string
	if len(meta.Director) > 0 {
		directorSubtitle = gettext.Get("Directed by") + " " + tagNames(meta.Director)
	}

	playLabel := gettext.Get("Play")
	if meta.ViewOffset > 0 {
		playLabel = fmt.Sprintf("%s %s", gettext.Get("Continue"), widgets.FormatTimestamp(meta.ViewOffset))
	}

	watchLabel := gettext.Get("Mark as Watched")
	watchTooltip := gettext.Get("Mark this movie as watched")
	if meta.ViewCount > 0 {
		watchLabel = gettext.Get("Mark as Unwatched")
		watchTooltip = gettext.Get("Mark this movie as unwatched")
	}

	heroContent := widgets.HeroContent(widgets.HeroContentParams{
		Title:      meta.Title,
		Subtitle:   directorSubtitle,
		Badges:     []string{fmt.Sprint(meta.Year), widgets.FormatDuration(meta.Duration), meta.ContentRating},
		Ratings:    meta.Ratings,
		UserRating: meta.UserRating,
		BuildButtonRow: func() schwifty.Box {
			return HStack().Spacing(10).
				Append(
					Button().
						Child(
							HStack(
								Image().FromIconName("media-playback-start-symbolic"),
								Label(playLabel),
							).Spacing(6),
						).
						TooltipText(gettext.Get("Play this movie")).
						WithCSSClass("suggested-action").
						WithCSSClass("pill").
						ConnectClicked(func(b gtk.Button) {
							if len(meta.Media) > 0 && len(meta.Media[0].Part) > 0 {
								player.NewPlayer(player.PlayerParams{
									Ctx:       appCtx.Ctx,
									Title:     meta.Title,
									PartKey:   meta.Media[0].Part[0].Key,
									Window:    appCtx.Window,
									RatingKey: ratingKey,
									Media:     meta.Media,
									Source:    src,
								})
							}
						}),
				).
				Append(
					Button().
						Child(
							HStack(
								Image().FromIconName("check-plain-symbolic"),
								Label(watchLabel),
							).Spacing(6),
						).
						TooltipText(watchTooltip).
						WithCSSClass("pill").
						ConnectClicked(func(b gtk.Button) {
							b.SetSensitive(false)
							watched := meta.ViewCount > 0
							go func() {
								var err error
								if watched {
									err = src.Unscrobble(appCtx.Ctx, ratingKey)
								} else {
									err = src.Scrobble(appCtx.Ctx, ratingKey)
								}
								if err != nil {
									slog.Error("failed to update watch status", "ratingKey", ratingKey, "error", err)
									schwifty.OnMainThreadOncePure(func() {
										b.SetSensitive(true)
										notifications.OnToast.Notify(gettext.Get("Failed to update watch status"))
									})
									return
								}
								schwifty.OnMainThreadOncePure(func() {
									b.SetSensitive(true)
									if watched {
										meta.ViewCount = 0
										b.SetTooltipText(gettext.Get("Mark this movie as watched"))
										b.SetChild(HStack(
											Image().FromIconName("check-plain-symbolic"),
											Label(gettext.Get("Mark as Watched")),
										).Spacing(6).ToGTK())
										notifications.OnToast.Notify(gettext.Get("Marked as unwatched"))
									} else {
										meta.ViewCount = 1
										b.SetTooltipText(gettext.Get("Mark this movie as unwatched"))
										b.SetChild(HStack(
											Image().FromIconName("check-plain-symbolic"),
											Label(gettext.Get("Mark as Unwatched")),
										).Spacing(6).ToGTK())
										notifications.OnToast.Notify(gettext.Get("Marked as watched"))
									}
								})
							}()
						}),
				)
		},
		Tagline: meta.Tagline,
		Summary: meta.Summary,
		MetadataRows: []widgets.MetadataRow{
			{Label: "Genres", Value: tagNames(meta.Genre)},
			{Label: "Writers", Value: tagNames(meta.Writer)},
			{Label: "Studio", Value: meta.Studio},
		},
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

	// Media info section
	if infoCards := cards.MediaInfo(meta.Media); infoCards != nil {
		body = body.Append(infoCards)
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

func tagNames(tags []sources.Tag) string {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Tag
	}
	return strings.Join(names, ", ")
}
