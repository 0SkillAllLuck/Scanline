package player

import (
	"context"
	"log/slog"

	"github.com/0skillallluck/scanline/app/sources"
)

// ResolveNextEpisode finds the episode after currentEp.
// It checks the same season first, then the next season of the same show.
// Returns nil if there is no next episode (last episode of the show).
func ResolveNextEpisode(ctx context.Context, src sources.Source, currentEp *sources.Metadata) *NextEpisodeInfo {
	if currentEp == nil || currentEp.Type != "episode" {
		return nil
	}

	// Try to find the next episode in the same season.
	siblings, err := src.GetChildren(ctx, currentEp.ParentRatingKey)
	if err != nil {
		slog.Debug("next episode: failed to fetch siblings", "error", err)
		return nil
	}

	for i := range siblings {
		if siblings[i].Index == currentEp.Index+1 {
			ep := &siblings[i]
			return metadataToNextInfo(ep)
		}
	}

	// Last episode of the season — try the next season.
	if currentEp.GrandparentRatingKey == "" {
		return nil
	}

	seasons, err := src.GetChildren(ctx, currentEp.GrandparentRatingKey)
	if err != nil {
		slog.Debug("next episode: failed to fetch seasons", "error", err)
		return nil
	}

	for i := range seasons {
		if seasons[i].Index == currentEp.ParentIndex+1 && seasons[i].Index != 0 {
			episodes, err := src.GetChildren(ctx, seasons[i].RatingKey)
			if err != nil || len(episodes) == 0 {
				return nil
			}
			ep := &episodes[0]
			return metadataToNextInfo(ep)
		}
	}

	return nil
}

func metadataToNextInfo(ep *sources.Metadata) *NextEpisodeInfo {
	if len(ep.Media) == 0 || len(ep.Media[0].Part) == 0 {
		return nil
	}
	return &NextEpisodeInfo{
		Title:      ep.Title,
		PartKey:    ep.Media[0].Part[0].Key,
		RatingKey:  ep.RatingKey,
		Media:      ep.Media,
		ViewOffset: ep.ViewOffset,
		Metadata:   ep,
	}
}
