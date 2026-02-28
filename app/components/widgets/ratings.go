package widgets

import (
	"fmt"
	"strings"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/gdk"
)

// RatingsParams contains the rating values for the Ratings component.
type RatingsParams struct {
	Ratings    sources.Ratings // All ratings from various sources
	UserRating float64         // User's personal rating
}

// ratingIconPath returns the resource path for a rating image string.
func ratingIconPath(ratingImage string) string {
	switch {
	case strings.Contains(ratingImage, "rottentomatoes://"):
		// Check if it's an audience rating (popcorn) or critic rating (tomato)
		if strings.Contains(ratingImage, "upright") || strings.Contains(ratingImage, "spilled") {
			return "/dev/skillless/Scanline/icons/scalable/ratings/rt-popcorn.svg"
		}
		return "/dev/skillless/Scanline/icons/scalable/ratings/rt-tomato.svg"
	case strings.Contains(ratingImage, "imdb://"):
		return "/dev/skillless/Scanline/icons/scalable/ratings/imdb.svg"
	case strings.Contains(ratingImage, "themoviedb://"):
		return "/dev/skillless/Scanline/icons/scalable/ratings/tmdb.svg"
	default:
		return ""
	}
}

// Ratings creates a horizontal row of rating badges with icons.
// Returns nil if no ratings are available.
func Ratings(params RatingsParams) schwifty.Box {
	ratings := HStack().Spacing(16)
	hasRatings := false

	// Process all ratings from sources
	for _, r := range params.Ratings {
		if r.Value <= 0 {
			continue
		}

		iconPath := ratingIconPath(r.Image)
		var icon schwifty.Image
		if iconPath != "" {
			texture := gdk.NewTextureFromResource(iconPath)
			icon = Image().FromPaintable(texture).PixelSize(16)
		} else {
			// Fallback icons based on type
			if r.Type == "critic" {
				icon = Image().FromIconName("starred-symbolic").PixelSize(16)
			} else {
				icon = Image().FromIconName("people-symbolic").PixelSize(16)
			}
		}

		// Format the rating value
		var label string
		if r.Value >= 10 {
			// Percentage-style rating (like RT)
			label = fmt.Sprintf("%.0f%%", r.Value*10)
		} else {
			// 10-point scale rating
			label = fmt.Sprintf("%.1f", r.Value)
		}

		ratings = ratings.Append(
			HStack(
				icon,
				Label(label),
			).Spacing(4),
		)
		hasRatings = true
	}

	// User rating
	if params.UserRating > 0 {
		ratings = ratings.Append(
			HStack(
				Image().FromIconName("star-filled-symbolic").PixelSize(16),
				Label(fmt.Sprintf("%.1f", params.UserRating)),
			).Spacing(4),
		)
		hasRatings = true
	}

	if !hasRatings {
		return nil
	}
	return ratings
}
