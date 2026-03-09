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
	var personName, personThumb string
	seen := make(map[string]bool)

	// Search across actor, director, and writer credits.
	filters := []sources.ContentOptions{
		{Actor: tagID},
		{Director: tagID},
		{Writer: tagID},
	}

	for _, section := range sections {
		if section.Type != "movie" && section.Type != "show" {
			continue
		}

		for i := range filters {
			content, _, err := src.LibraryContent(ctx, section.Key, &filters[i])
			if err != nil {
				slog.Debug("failed to fetch content for section", "section", section.Key, "error", err)
				continue
			}
			for _, item := range content {
				if !seen[item.RatingKey] {
					seen[item.RatingKey] = true
					allContent = append(allContent, item)
				}
			}
		}
	}

	// The listing endpoint doesn't include tag data, so fetch full metadata
	// for the first result to extract the person's name and thumbnail.
	if len(allContent) > 0 {
		tagIDInt, _ := strconv.Atoi(tagID)
		meta, err := src.GetMetadata(ctx, allContent[0].RatingKey)
		if err == nil {
			// Search across all credit types for the matching tag ID.
			for _, tags := range [][]sources.Tag{meta.Role, meta.Director, meta.Writer} {
				for _, tag := range tags {
					if tag.ID == tagIDInt {
						personName = tag.Tag
						personThumb = tag.Thumb
						break
					}
				}
				if personName != "" {
					break
				}
			}
		}
	}

	if personName == "" {
		personName = gettext.Get("Unknown Actor")
	}

	body := VStack().Spacing(20).VMargin(20).HMargin(20)

	// Actor header: circular photo + name
	header := VStack().HAlign(gtk.AlignCenterValue).Spacing(12).MarginBottom(10)
	if personThumb != "" {
		header = header.Append(
			Bin().
				Child(
					Picture().
						SizeRequest(200, 200).
						ContentFit(gtk.ContentFitCoverValue).
						ConnectRealize(func(w gtk.Widget) {
							if preference.Performance().AllowPreviewImages() {
								imageutils.LoadIntoPictureCropped(src.PhotoTranscodeURL(personThumb, 200, 200), 200, gtk.PictureNewFromInternalPtr(w.Ptr))
							}
						}),
				).
				SizeRequest(200, 200).
				CornerRadius(100).
				Overflow(gtk.OverflowHiddenValue),
		)
	}
	header = header.Append(
		Label(personName).
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
		if card, ok := lists.MetadataCard(meta, coverURL, personName, serverID); ok {
			grid = grid.Append(card)
		}
	}

	body = body.Append(grid)

	return &router.Response{
		PageTitle: personName,
		View: ScrolledWindow().
			Child(Clamp().MaximumSize(1200).Child(body)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}
