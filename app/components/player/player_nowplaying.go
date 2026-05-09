package player

import (
	"context"
	"log/slog"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	"github.com/0skillallluck/scanline/app/components/player/nowplaying"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

// nowPlayingArtworkSize is the square edge length we request from the source's
// photo transcoder. Control Center crops to a square thumbnail; 600px is a
// reasonable balance between memory and Retina sharpness.
const nowPlayingArtworkSize = 600

// nowplayingKindFor guesses whether the playing item is a movie or an episode
// from the params we already have at session start. The async metadata fetch
// later refines this if needed.
func nowplayingKindFor(params PlayerParams) nowplaying.MediaKind {
	if params.GrandparentRatingKey != "" {
		return nowplaying.KindEpisode
	}
	return nowplaying.KindMovie
}

// fetchAndPushArtwork runs in a goroutine on session start. It fetches the
// full metadata (for show/season titles and the artwork URL), pulls the
// artwork bytes through the existing image cache, and hops to the GTK main
// thread to push refined text + artwork into MPNowPlayingInfoCenter.
//
// On non-Darwin platforms the SetTextMetadata / SetArtwork calls are no-ops,
// so the metadata + image HTTP fetches are wasted but harmless.
func fetchAndPushArtwork(ctx context.Context, src sources.Source, params PlayerParams) {
	if ctx.Err() != nil {
		return
	}
	meta, err := src.GetMetadata(ctx, params.RatingKey)
	if err != nil || meta == nil {
		slog.Warn("nowplaying: metadata fetch failed", "error", err)
		return
	}
	if ctx.Err() != nil {
		return
	}

	info := nowplaying.Info{
		Title:      params.Title,
		Kind:       nowplayingKindFor(params),
		DurationUs: int64(meta.Duration) * 1000,
	}
	if meta.Type == "episode" {
		info.Kind = nowplaying.KindEpisode
		info.Artist = meta.GrandparentTitle
		info.AlbumTitle = meta.ParentTitle
	}
	schwifty.OnMainThreadOncePure(func() {
		if ctx.Err() != nil {
			return
		}
		nowplaying.SetTextMetadata(info)
	})

	artURL := bestArtURL(meta)
	if artURL == "" {
		return
	}
	transcoded := src.PhotoTranscodeURL(artURL, nowPlayingArtworkSize, nowPlayingArtworkSize)
	data, err := imageutils.Fetch(transcoded)
	if err != nil {
		slog.Warn("nowplaying: artwork fetch failed", "error", err)
		return
	}
	if ctx.Err() != nil {
		return
	}
	schwifty.OnMainThreadOncePure(func() {
		if ctx.Err() != nil {
			return
		}
		nowplaying.SetArtwork(data)
	})
}

// bestArtURL picks the most square-friendly artwork URL for Now Playing.
// Episodes prefer the show poster (GrandparentThumb); movies prefer Thumb;
// otherwise we fall back to the generic ArtURL helper.
func bestArtURL(meta *sources.Metadata) string {
	if meta.Type == "episode" && meta.GrandparentThumb != "" {
		return meta.GrandparentThumb
	}
	if meta.Thumb != "" {
		return meta.Thumb
	}
	return sources.ArtURL(meta)
}
