package pages

import (
	"log/slog"
	"strconv"

	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/adw"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

var CastRoute = router.NewRoute("cast/:server/:tagId", Cast)

func Cast(appCtx *appctx.AppContext, serverID, tagID string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("Cast"), errSourceNotFound(serverID))
	}

	ctx := appCtx.Ctx

	sections, err := src.LibrarySections(ctx)
	if err != nil {
		return router.FromError(gettext.Get("Cast"), err)
	}

	coverURL := func(thumb string) string {
		return src.PhotoTranscodeURL(thumb, 240, 360)
	}

	var allContent []sources.Metadata
	var actorName, actorThumb string

	for _, section := range sections {
		if section.Type != "movie" && section.Type != "show" {
			continue
		}

		content, _, err := src.LibraryContent(ctx, section.Key, &sources.ContentOptions{Actor: tagID})
		if err != nil {
			slog.Debug("failed to fetch content for section", "section", section.Key, "error", err)
			continue
		}

		allContent = append(allContent, content...)
	}

	// The listing endpoint doesn't include Role data, so fetch full metadata
	// for the first result to extract the actor name and thumbnail.
	if len(allContent) > 0 {
		tagIDInt, _ := strconv.Atoi(tagID)
		meta, err := src.GetMetadata(ctx, allContent[0].RatingKey)
		if err == nil {
			for _, role := range meta.Role {
				if role.ID == tagIDInt {
					actorName = role.Tag
					actorThumb = role.Thumb
					break
				}
			}
		}
	}

	if actorName == "" {
		actorName = gettext.Get("Unknown Actor")
	}

	body := VStack().Spacing(20).VMargin(20).HMargin(20)

	// Actor header: circular photo + name
	header := VStack().HAlign(gtk.AlignCenterValue).Spacing(12).MarginBottom(10)
	if actorThumb != "" {
		header = header.Append(
			Bin().
				Child(
					Picture().
						SizeRequest(200, 200).
						ContentFit(gtk.ContentFitCoverValue).
						ConnectRealize(func(w gtk.Widget) {
							if preference.Performance().AllowPreviewImages() {
								imageutils.LoadIntoPictureCropped(src.PhotoTranscodeURL(actorThumb, 200, 200), 200, gtk.PictureNewFromInternalPtr(w.Ptr))
							}
						}),
				).
				SizeRequest(200, 200).
				CornerRadius(100).
				Overflow(gtk.OverflowHiddenValue),
		)
	}
	header = header.Append(
		Label(actorName).
			WithCSSClass("title-1").
			HAlign(gtk.AlignCenterValue).
			Ellipsis(pango.EllipsizeEndValue),
	)
	body = body.Append(header)

	// Content grid
	grid := WrapBox().
		ConnectConstruct(func(w *adw.WrapBox) {
			w.SetChildSpacing(20)
			w.SetLineSpacing(20)
			w.SetLineHomogeneous(true)
			w.SetJustify(adw.JustifyFillValue)
		})

	for i := range allContent {
		meta := &allContent[i]
		if card, ok := lists.MetadataCard(meta, coverURL, actorName, serverID); ok {
			grid = grid.Append(card)
		}
	}

	body = body.Append(grid)

	return &router.Response{
		PageTitle: actorName,
		View: ScrolledWindow().
			Child(Clamp().MaximumSize(1200).Child(body)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}
