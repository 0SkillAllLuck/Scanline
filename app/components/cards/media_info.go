package cards

import (
	"fmt"
	"strings"

	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

// MediaInfo creates an HStack of info cards showing resolution, video codec, audio, and container.
// Returns nil if no media info is available.
func MediaInfo(media []sources.Media) schwifty.Box {
	if len(media) == 0 {
		return nil
	}

	m := media[0]

	// Collect info card data
	type cardData struct {
		icon, title, subtitle string
	}
	var cardItems []cardData

	if m.VideoResolution != "" {
		cardItems = append(cardItems, cardData{
			"display-symbolic",
			strings.ToUpper(m.VideoResolution),
			gettext.Get("Resolution"),
		})
	}
	if m.VideoCodec != "" {
		cardItems = append(cardItems, cardData{
			"filmstrip-symbolic",
			formatCodec(m.VideoCodec),
			gettext.Get("Video"),
		})
	}
	if m.AudioCodec != "" {
		audioLabel := formatCodec(m.AudioCodec)
		if m.AudioChannels > 0 {
			audioLabel += " " + formatChannels(m.AudioChannels)
		}
		cardItems = append(cardItems, cardData{
			"audio-speakers-symbolic",
			audioLabel,
			gettext.Get("Audio"),
		})
	}
	if m.Container != "" {
		cardItems = append(cardItems, cardData{
			"package-x-generic-symbolic",
			strings.ToUpper(m.Container),
			gettext.Get("Container"),
		})
	}

	if len(cardItems) == 0 {
		return nil
	}

	// Build HStack with separators between cards
	infoCards := HStack().VAlign(gtk.AlignStartValue)

	for i, item := range cardItems {
		if i > 0 {
			// Add separator before each card except the first
			infoCards = infoCards.Append(
				VStack().
					CSS("box { background: alpha(currentColor, 0.15); }").
					MinWidth(1),
			)
		}
		infoCards = infoCards.Append(NewInfoCard(item.icon, item.title, item.subtitle))
	}

	// Wrap in a card container
	return HStack(infoCards).
		WithCSSClass("card").
		VExpand(false).
		VAlign(gtk.AlignStartValue).
		Padding(15)
}

// formatCodec returns a human-readable codec name.
func formatCodec(codec string) string {
	switch strings.ToLower(codec) {
	case "h264":
		return "H.264"
	case "hevc", "h265":
		return "HEVC"
	case "vp9":
		return "VP9"
	case "av1":
		return "AV1"
	case "aac":
		return "AAC"
	case "ac3":
		return "AC3"
	case "eac3":
		return "EAC3"
	case "dts":
		return "DTS"
	case "flac":
		return "FLAC"
	case "mp3":
		return "MP3"
	default:
		return strings.ToUpper(codec)
	}
}

// formatChannels returns a human-readable channel configuration.
func formatChannels(channels int) string {
	switch channels {
	case 1:
		return "Mono"
	case 2:
		return "Stereo"
	case 6:
		return "5.1"
	case 8:
		return "7.1"
	default:
		return fmt.Sprintf("%dch", channels)
	}
}
