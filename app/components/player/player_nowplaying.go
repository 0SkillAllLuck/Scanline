package player

import "github.com/0skillallluck/scanline/app/components/player/nowplaying"

// nowplayingKindFor guesses whether the playing item is a movie or an episode
// from the params we already have at session start. The async metadata fetch
// later refines this if needed.
func nowplayingKindFor(params PlayerParams) nowplaying.MediaKind {
	if params.GrandparentRatingKey != "" {
		return nowplaying.KindEpisode
	}
	return nowplaying.KindMovie
}
