package search

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

func Results(hubs []sources.Hub, coverURL func(string) string, serverID string) schwifty.Box {
	body := VStack().Spacing(25).VMargin(20)
	for i := range hubs {
		hub := &hubs[i]
		list := lists.NewHorizontalList(hub.Title)
		if lists.RenderHub(list, hub, coverURL, serverID) {
			body = body.Append(list.SetPageMargin(40))
		}
	}
	return body.VAlign(gtk.AlignStartValue)
}
