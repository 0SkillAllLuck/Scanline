package widgets

import (
	"fmt"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/internal/resources"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/utils/imageutils"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/gtk"
	"github.com/jwijenbergh/puregotk/v4/pango"
)

// HeroPosterParams configures the poster image in a hero section.
type HeroPosterParams struct {
	ImageURL string
	Width    int
	Height   int
}

// HeroSection creates a hero layout with a poster image and content area.
// The content parameter is positioned to the right of the poster.
func HeroSection(poster HeroPosterParams, content schwifty.Box) schwifty.Box {
	cssSize := fmt.Sprintf(
		"picture { min-width: %dpx; min-height: %dpx; }",
		poster.Width, poster.Height,
	)

	return HStack(
		Picture().
			SizeRequest(poster.Width, poster.Height).
			ContentFit(gtk.ContentFitCoverValue).
			FromPaintable(resources.MissingAlbum()).
			ConnectRealize(func(w gtk.Widget) {
				if preference.Performance().AllowPreviewImages() {
					imageutils.LoadIntoPictureScaled(poster.ImageURL, poster.Width, poster.Height, gtk.PictureNewFromInternalPtr(w.Ptr))
				}
			}).
			CSS(cssSize).
			CornerRadius(10).Overflow(gtk.OverflowHiddenValue),
		content.VAlign(gtk.AlignStartValue).MarginStart(24),
	).Spacing(0)
}

// MetadataRow represents a labeled metadata item (e.g., "Genres: Action, Drama").
type MetadataRow struct {
	Label string
	Value string
}

// HeroContentParams configures the hero content section.
type HeroContentParams struct {
	Title          string              // Main title (required)
	TitleClass     string              // CSS class (default: "title-1")
	Subtitle       string              // Secondary text (optional)
	SubtitleClass  string              // CSS class (default: "dimmed")
	Badges         []string            // Meta badges: year, duration, etc.
	Ratings        sources.Ratings     // Ratings from various sources
	UserRating     float64             // User's personal rating
	BuildButtonRow func() schwifty.Box // Function that builds the button row (optional)
	Tagline        string              // Bold tagline (optional)
	Summary        string              // Description text (optional)
	MetadataRows   []MetadataRow       // Label: Value pairs
}

// HeroContent creates the content portion for a hero section.
func HeroContent(params HeroContentParams) schwifty.Box {
	content := VStack().Spacing(6).HAlign(gtk.AlignStartValue)

	// Title
	titleClass := params.TitleClass
	if titleClass == "" {
		titleClass = "title-1"
	}
	content = content.Append(
		Label(params.Title).
			WithCSSClass(titleClass).
			HAlign(gtk.AlignStartValue).
			Wrap(true).
			WrapMode(pango.WrapWordCharValue),
	)

	// Subtitle
	if params.Subtitle != "" {
		subtitleClass := params.SubtitleClass
		if subtitleClass == "" {
			subtitleClass = "dimmed"
		}
		subtitle := Label(params.Subtitle).
			HAlign(gtk.AlignStartValue).
			Wrap(true).
			WrapMode(pango.WrapWordCharValue)
		// Apply all CSS classes (supports space-separated classes like "title-2 dimmed")
		for _, class := range splitClasses(subtitleClass) {
			subtitle = subtitle.WithCSSClass(class)
		}
		content = content.Append(subtitle)
	}

	// Badges row
	if len(params.Badges) > 0 {
		badges := HStack().Spacing(12)
		for _, badge := range params.Badges {
			if badge != "" {
				badges = badges.Append(
					Label(badge).WithCSSClass("dimmed"),
				)
			}
		}
		content = content.Append(badges.MarginTop(4))
	}

	// Ratings row
	if ratings := Ratings(RatingsParams{
		Ratings:    params.Ratings,
		UserRating: params.UserRating,
	}); ratings != nil {
		content = content.Append(ratings.MarginTop(4))
	}

	// Action buttons
	if params.BuildButtonRow != nil {
		content = content.Append(params.BuildButtonRow().MarginTop(8))
	}

	// Tagline (bold)
	if params.Tagline != "" {
		content = content.Append(
			Label(params.Tagline).
				WithCSSClass("heading").
				HAlign(gtk.AlignStartValue).
				Wrap(true).
				MarginTop(8),
		)
	}

	// Summary
	if params.Summary != "" {
		content = content.Append(
			Label(params.Summary).
				HAlign(gtk.AlignStartValue).
				Wrap(true).
				WrapMode(pango.WrapWordCharValue).
				MarginTop(8),
		)
	}

	// Metadata rows
	for i, row := range params.MetadataRows {
		if row.Value == "" {
			continue
		}
		marginTop := 4
		if i == 0 {
			marginTop = 8
		}
		content = content.Append(
			HStack(
				Label(gettext.Get(row.Label)+":").WithCSSClass("dimmed"),
				Label(row.Value),
			).Spacing(6).HAlign(gtk.AlignStartValue).MarginTop(marginTop),
		)
	}

	return content
}

// splitClasses splits a space-separated class string into individual classes.
func splitClasses(classes string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(classes); i++ {
		if i == len(classes) || classes[i] == ' ' {
			if i > start {
				result = append(result, classes[start:i])
			}
			start = i + 1
		}
	}
	return result
}

// FormatDuration formats milliseconds as "2h 30m" or "45m".
func FormatDuration(ms int) string {
	minutes := ms / 60000
	h := minutes / 60
	m := minutes % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// FormatSeasonCount returns a pluralized season count string.
func FormatSeasonCount(count int) string {
	return gettext.GetN("%d Season", "%d Seasons", count, count)
}

// FormatEpisodeLabel formats a season and episode number as "S2 - E5".
func FormatEpisodeLabel(season, episode int) string {
	return fmt.Sprintf("S%d - E%d", season, episode)
}

// FormatTimestamp formats milliseconds as "H:MM:SS" or "M:SS".
func FormatTimestamp(ms int) string {
	totalSeconds := ms / 1000
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
