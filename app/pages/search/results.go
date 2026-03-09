package search

import (
	"cmp"
	"slices"
	"strings"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/components/lists"
	"github.com/0skillallluck/scanline/app/sources"
)

const (
	SortTitleAsc = iota
	SortTitleDesc
	SortYearDesc
	SortYearAsc
	SortRecentlyAdded
)

// SortHubs sorts the Metadata slice within each hub in-place.
func SortHubs(hubs []sources.Hub, sortOption int) {
	for i := range hubs {
		slices.SortStableFunc(hubs[i].Metadata, func(a, b sources.Metadata) int {
			switch sortOption {
			case SortTitleDesc:
				return cmp.Compare(strings.ToLower(titleKey(b)), strings.ToLower(titleKey(a)))
			case SortYearDesc:
				return cmp.Compare(b.Year, a.Year)
			case SortYearAsc:
				return cmp.Compare(a.Year, b.Year)
			case SortRecentlyAdded:
				return cmp.Compare(b.AddedAt, a.AddedAt)
			default: // SortTitleAsc
				return cmp.Compare(strings.ToLower(titleKey(a)), strings.ToLower(titleKey(b)))
			}
		})
	}
}

func titleKey(m sources.Metadata) string {
	if m.TitleSort != "" {
		return m.TitleSort
	}
	return m.Title
}

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
