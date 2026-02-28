package lists

import (
	"log/slog"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/components/cards"
	"github.com/0skillallluck/scanline/app/sources"
)

// MetadataCard creates the appropriate card widget for the given metadata type.
// Returns the card widget and true if successful, nil and false for unsupported types.
func MetadataCard(meta *sources.Metadata, coverURL func(string) string, context, serverID string) (schwifty.BaseWidgetable, bool) {
	switch meta.Type {
	case "movie":
		return cards.NewMoviePoster(meta, coverURL(meta.Thumb), serverID), true
	case "show":
		return cards.NewShowPoster(meta, meta.ChildCount, coverURL(meta.Thumb), serverID), true
	case "episode":
		return cards.NewEpisodePoster(meta, coverURL(meta.GrandparentThumb), serverID), true
	default:
		slog.Debug("unsupported metadata type", "type", meta.Type, "context", context)
		return nil, false
	}
}

// RenderHubMetadata appends the appropriate card type to the list based on metadata type.
// Returns true if an item was added, false otherwise.
func RenderHubMetadata(list *HorizontalList, meta *sources.Metadata, coverURL func(string) string, hubTitle, serverID string) bool {
	if card, ok := MetadataCard(meta, coverURL, hubTitle, serverID); ok {
		list.Append(card)
		return true
	}
	return false
}

// RenderHub renders all metadata items in a hub to the given list.
// Returns true if at least one item was added.
func RenderHub(list *HorizontalList, hub *sources.Hub, coverURL func(string) string, serverID string) bool {
	hasItems := false
	for i := range hub.Metadata {
		meta := &hub.Metadata[i]
		if RenderHubMetadata(list, meta, coverURL, hub.Title, serverID) {
			hasItems = true
		}
	}
	return hasItems
}
