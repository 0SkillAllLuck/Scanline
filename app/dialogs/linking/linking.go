package linking

import (
	"bytes"
	"context"
	"log/slog"
	"time"

	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/0skillallluck/scanline/internal/resources"
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gdk"
	"github.com/jwijenbergh/puregotk/v4/glib"
	"github.com/jwijenbergh/puregotk/v4/gtk"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"github.com/yeqown/go-qrcode/writer/standard/shapes"
)

var QRCode = Image().
	PixelSize(186).
	FromPaintable(resources.MissingAlbum())

var Helper = Label(gettext.Get("You can also open the linking page using the button below."))

type QRBuffer struct {
	bytes.Buffer
}

func (q *QRBuffer) Close() error {
	return nil
}

func NewLinking(window *gtk.Window, authURL string, cancel context.CancelFunc) schwifty.AlertDialog {
	encodedUrl, err := qrcode.New(authURL)
	if err != nil {
		slog.Error("could not generate QR code to sign in", "error", err)
	}

	var buf QRBuffer
	shape := shapes.Assemble(shapes.RoundedFinder(), shapes.LiquidBlock())
	writer := standard.NewWithWriter(&buf, standard.WithCustomShape(shape))

	if err := encodedUrl.Save(writer); err != nil {
		slog.Error("could not write QR code to sign in", "error", err)
	}

	gBytes := glib.NewBytes(buf.Bytes(), uint(buf.Len()))
	texture, err := gdk.NewTextureFromBytes(gBytes)
	if err != nil {
		slog.Error("could not create texture from bytes", "error", err)
	}

	return AlertDialog(gettext.Get("Sign In"), gettext.Get("Scan this QR code to sign into your Plex account.")).
		WithCSSClass("no-response").
		CanClose(false).
		ConnectCloseAttempt(func(d adw.Dialog) {
			cancel()
		}).
		ExtraChild(
			VStack(
				AspectFrame(
					QRCode.FromPaintable(texture),
				).Background("alpha(var(--view-fg-color), 0.1)").HAlign(gtk.AlignCenterValue).CornerRadius(10).
					Overflow(gtk.OverflowHiddenValue),
				Helper,
				VStack(
					Button().
						Label(gettext.Get("Open Plex page")).
						WithCSSClass("suggested-action").
						HPadding(20).VPadding(10).
						ConnectClicked(func(b gtk.Button) {
							gtk.ShowUri(window, authURL, uint32(time.Now().Unix()))
						}),
					Button().
						Label(gettext.Get("Cancel Login")).
						WithCSSClass("destructive-action").
						HPadding(20).VPadding(10).
						ConnectClicked(func(b gtk.Button) {
							cancel()
						}),
				).Spacing(10),
			).Spacing(20),
		)
}
