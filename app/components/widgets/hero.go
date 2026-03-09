package widgets

import (
	"fmt"
	"strconv"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/utils/imageutils"
)

// linkButtonCSS strips button chrome so it looks like inline text.
const linkButtonCSS = `button {
	background: none;
	border: none;
	box-shadow: none;
	padding: 0;
	min-height: 0;
	min-width: 0;
	font-weight: inherit;
}`

// linkButton creates a button styled as a text link with pointer cursor
// and underline on hover.
func linkButton(label schwifty.Label, actionName, actionValue string) schwifty.Button {
	underlineAttrs := pango.NewAttrList()
	underlineAttrs.Insert(pango.AttrUnderlineNew(pango.UnderlineSingleValue))

	var labelPtr uintptr
	label = label.ConnectConstruct(func(l *gtk.Label) {
		labelPtr = l.GoPointer()
	})

	hover := gtk.NewEventControllerMotion()
	hover.ConnectEnter(new(func(gtk.EventControllerMotion, float64, float64) {
		gtk.LabelNewFromInternalPtr(labelPtr).SetAttributes(underlineAttrs)
	}))
	hover.ConnectLeave(new(func(gtk.EventControllerMotion) {
		gtk.LabelNewFromInternalPtr(labelPtr).SetAttributes(nil)
	}))

	return Button().
		Child(label).
		WithCSSClass("flat").
		CSS(linkButtonCSS).
		HAlign(gtk.AlignStartValue).
		ActionName(actionName).
		ActionTargetValue(glib.NewVariantString(actionValue)).
		AddController(&hover.EventController).
		ConnectRealize(func(w gtk.Widget) {
			w.SetCursorFromName("pointer")
		})
}

// BadgeLink is a clickable badge that triggers a GTK action.
type BadgeLink struct {
	Label       string
	ActionName  string
	ActionValue string
}

// HeroPosterParams configures the poster image in a hero section.
type HeroPosterParams struct {
	ImageURL string
	Width    int32
	Height   int32
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
			FromPaintable(gdk.NewTextureFromResource("/dev/skillless/Scanline/icons/scalable/state/missing-album.svg")).
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
// When Tags and ServerID are set, each tag is rendered as a clickable link to the cast page.
type MetadataRow struct {
	Label    string
	Value    string
	Tags     []sources.Tag
	ServerID string
}

// HeroContentParams configures the hero content section.
type HeroContentParams struct {
	Title            string              // Main title (required)
	TitleClass       string              // CSS class (default: "title-1")
	TitleActionName  string              // GTK action name to trigger on click (optional, makes title clickable)
	TitleActionValue string              // GTK action target value (optional)
	Subtitle            string              // Secondary text (optional)
	SubtitleClass       string              // CSS class (default: "dimmed")
	SubtitleActionName  string              // GTK action name to trigger on click (optional, makes subtitle clickable)
	SubtitleActionValue string              // GTK action target value (optional)
	BadgeLinks     []BadgeLink          // Clickable badges with navigation (rendered before Badges)
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
	if params.TitleActionName != "" {
		titleLabel := Label(params.Title).
			WithCSSClass(titleClass).
			Wrap(true).
			WrapMode(pango.WrapWordCharValue)
		content = content.Append(linkButton(titleLabel, params.TitleActionName, params.TitleActionValue))
	} else {
		content = content.Append(
			Label(params.Title).
				WithCSSClass(titleClass).
				HAlign(gtk.AlignStartValue).
				Wrap(true).
				WrapMode(pango.WrapWordCharValue),
		)
	}

	// Subtitle
	if params.Subtitle != "" {
		subtitleClass := params.SubtitleClass
		if subtitleClass == "" {
			subtitleClass = "dimmed"
		}
		if params.SubtitleActionName != "" {
			subtitleLabel := Label(params.Subtitle).
				Wrap(true).
				WrapMode(pango.WrapWordCharValue)
			for _, class := range splitClasses(subtitleClass) {
				subtitleLabel = subtitleLabel.WithCSSClass(class)
			}
			content = content.Append(linkButton(subtitleLabel, params.SubtitleActionName, params.SubtitleActionValue))
		} else {
			subtitle := Label(params.Subtitle).
				HAlign(gtk.AlignStartValue).
				Wrap(true).
				WrapMode(pango.WrapWordCharValue)
			for _, class := range splitClasses(subtitleClass) {
				subtitle = subtitle.WithCSSClass(class)
			}
			content = content.Append(subtitle)
		}
	}

	// Badges row
	if len(params.BadgeLinks) > 0 || len(params.Badges) > 0 {
		badges := HStack().Spacing(12)
		for _, bl := range params.BadgeLinks {
			badgeLabel := Label(bl.Label).WithCSSClass("dimmed")
			badges = badges.Append(linkButton(badgeLabel, bl.ActionName, bl.ActionValue))
		}
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
		if row.Value == "" && len(row.Tags) == 0 {
			continue
		}
		var marginTop int32 = 4
		if i == 0 {
			marginTop = 8
		}
		rowBox := HStack(
			Label(gettext.Get(row.Label) + ":").WithCSSClass("dimmed"),
		).Spacing(6).HAlign(gtk.AlignStartValue).MarginTop(marginTop)

		if len(row.Tags) > 0 && row.ServerID != "" {
			for j, tag := range row.Tags {
				name := tag.Tag
				if j < len(row.Tags)-1 {
					name += ","
				}
				tagLabel := Label(name)
				rowBox = rowBox.Append(linkButton(tagLabel, "win.route.cast", row.ServerID+"/"+strconv.Itoa(tag.ID)))
			}
		} else {
			rowBox = rowBox.Append(Label(row.Value))
		}
		content = content.Append(rowBox)
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

// FormatSeasonLabel formats a season number as "S2".
func FormatSeasonLabel(season int) string {
	return fmt.Sprintf("S%d", season)
}

// FormatEpisodeOnlyLabel formats an episode number as "E5".
func FormatEpisodeOnlyLabel(episode int) string {
	return fmt.Sprintf("E%d", episode)
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
