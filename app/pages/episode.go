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
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/utils/notifications"
)

var EpisodeRoute = router.NewRoute("episode/:server/:ratingKey", Episode)

func Episode(ctx context.Context, appCtx *appctx.AppContext, serverID, ratingKey string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("Episode"), errSourceNotFound(serverID))
	}

	meta, err := src.GetMetadata(ctx, ratingKey)
	if err != nil {
		return router.FromError(gettext.Get("Episode"), err)
	}

	body := VStack().Spacing(25).VMargin(20).HMargin(40)

	// Hero section
	playLabel := gettext.Get("Play")
	if meta.ViewOffset > 0 {
		playLabel = fmt.Sprintf("%s %s", gettext.Get("Continue"), widgets.FormatTimestamp(meta.ViewOffset))
	}

	watchLabel := gettext.Get("Mark as Watched")
	watchTooltip := gettext.Get("Mark this episode as watched")
	if meta.ViewCount > 0 {
		watchLabel = gettext.Get("Mark as Unwatched")
		watchTooltip = gettext.Get("Mark this episode as unwatched")
	}

	heroContent := widgets.HeroContent(widgets.HeroContentParams{
		Title:               meta.Title,
		Subtitle:            meta.GrandparentTitle,
		SubtitleActionName:  "win.route.show",
		SubtitleActionValue: serverID + "/" + meta.GrandparentRatingKey,
		BadgeLinks: []widgets.BadgeLink{{
			Label:       widgets.FormatSeasonLabel(meta.ParentIndex),
			ActionName:  "win.route.season",
			ActionValue: serverID + "/" + meta.ParentRatingKey,
		}},
		Badges: []string{widgets.FormatEpisodeOnlyLabel(meta.Index), widgets.FormatDuration(meta.Duration), meta.ContentRating},
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
						TooltipText(gettext.Get("Play this episode")).
						WithCSSClass("suggested-action").
						WithCSSClass("pill").
						ConnectClicked(func(b gtk.Button) {
							if len(meta.Media) > 0 && len(meta.Media[0].Part) > 0 {
								nextEp := player.ResolveNextEpisode(ctx, src, meta)
								player.NewPlayer(player.PlayerParams{
									Ctx:         ctx,
									Title:       meta.Title,
									PartKey:     meta.Media[0].Part[0].Key,
									Window:      appCtx.Window,
									RatingKey:   ratingKey,
									Media:       meta.Media,
									Source:      src,
									ViewOffset:  meta.ViewOffset,
									NextEpisode: nextEp,
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
									err = src.Unscrobble(ctx, ratingKey)
								} else {
									err = src.Scrobble(ctx, ratingKey)
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
										b.SetTooltipText(gettext.Get("Mark this episode as watched"))
										b.SetChild(HStack(
											Image().FromIconName("check-plain-symbolic"),
											Label(gettext.Get("Mark as Watched")),
										).Spacing(6).ToGTK())
										notifications.OnToast.Notify(gettext.Get("Marked as unwatched"))
									} else {
										meta.ViewCount = 1
										b.SetTooltipText(gettext.Get("Mark this episode as unwatched"))
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
		Summary: meta.Summary,
		MetadataRows: []widgets.MetadataRow{
			{Label: "Genres", Tags: meta.Genre, ServerID: serverID, ActionName: "win.route.genre"},
			{Label: "Directors", Tags: meta.Director, ServerID: serverID},
			{Label: "Writers", Tags: meta.Writer, ServerID: serverID},
		},
	})

	hero := widgets.HeroSection(
		widgets.HeroPosterParams{
			ImageURL: src.PhotoTranscodeURL(meta.Thumb, 480, 270),
			Width:    480,
			Height:   270,
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
			castList.Append(cards.NewCastMember(role.Tag, role.Role, src.PhotoTranscodeURL(role.Thumb, 140, 140), serverID, strconv.Itoa(role.ID)))
		}
		body = body.Append(castList.SetPageMargin(0))
	}

	return &router.Response{
		PageTitle: meta.Title,
		View: ScrolledWindow().
			Child(Clamp().MaximumSize(1200).Child(body)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}
