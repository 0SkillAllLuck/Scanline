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
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
)

var GenreRoute = router.NewRoute("genre/:server/:genreId", Genre)

func Genre(appCtx *appctx.AppContext, serverID, genreID string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("Genre"), errSourceNotFound(serverID))
	}

	ctx := appCtx.Ctx

	sections, err := src.LibrarySections(ctx)
	if err != nil {
		return router.FromError(gettext.Get("Genre"), err)
	}

	coverURL := func(thumb string) string {
		return src.PhotoTranscodeURL(thumb, 240, 360)
	}

	var allContent []sources.Metadata
	seen := make(map[string]bool)

	filter := sources.ContentOptions{Genre: genreID}

	for _, section := range sections {
		if section.Type != "movie" && section.Type != "show" {
			continue
		}

		content, _, err := src.LibraryContent(ctx, section.Key, &filter)
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

	// The listing endpoint may not include genre tag IDs, so fetch full
	// metadata for the first result to extract the genre name.
	genreName := gettext.Get("Genre")
	if len(allContent) > 0 {
		genreIDInt, _ := strconv.Atoi(genreID)
		meta, err := src.GetMetadata(ctx, allContent[0].RatingKey)
		if err == nil {
			for _, g := range meta.Genre {
				if g.ID == genreIDInt {
					genreName = g.Tag
					break
				}
			}
		}
	}

	body := VStack().Spacing(20).VMargin(20).HMargin(20)

	// Genre header
	header := VStack().HAlign(gtk.AlignCenterValue).Spacing(12).MarginBottom(10)
	header = header.Append(
		Label(genreName).
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
		if card, ok := lists.MetadataCard(meta, coverURL, genreName, serverID); ok {
			grid = grid.Append(card)
		}
	}

	body = body.Append(grid)

	return &router.Response{
		PageTitle: genreName,
		View: ScrolledWindow().
			Child(Clamp().MaximumSize(1200).Child(body)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}
