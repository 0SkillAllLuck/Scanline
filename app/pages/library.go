package pages

import (
	"context"

	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var LibraryRoute = router.NewRoute("library/:server/:id", Library)

func Library(appCtx *appctx.AppContext, serverID, sectionID string) *router.Response {
	mgr := appCtx.Manager
	src := mgr.SourceForServer(serverID)
	if src == nil {
		return router.FromError(gettext.Get("Library"), errSourceNotFound(serverID))
	}

	ctx := context.Background()

	section, err := src.LibrarySection(ctx, sectionID)
	if err != nil {
		return router.FromError(gettext.Get("Library"), err)
	}

	content, _, err := src.LibraryContent(ctx, sectionID, &sources.ContentOptions{Sort: "titleSort"})
	if err != nil {
		return router.FromError(section.Title, err)
	}

	coverURL := func(thumb string) string {
		return src.PhotoTranscodeURL(thumb, 240, 360)
	}

	body := WrapBox().
		ConnectConstruct(func(w *adw.WrapBox) {
			w.SetChildSpacing(20)
			w.SetLineSpacing(20)
			w.SetLineHomogeneous(true)
			w.SetJustify(adw.JustifyFillValue)
		})

	for i := range content {
		meta := &content[i]
		if card, ok := lists.MetadataCard(meta, coverURL, section.Title, serverID); ok {
			body = body.Append(card)
		}
	}

	return &router.Response{
		PageTitle: section.Title,
		View: ScrolledWindow().
			Child(body.VMargin(20).HMargin(20)).
			Policy(gtk.PolicyNeverValue, gtk.PolicyAutomaticValue),
	}
}
