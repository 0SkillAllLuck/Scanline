package gettext

import (
	"log/slog"

	"github.com/0skillallluck/scanline/locales"
	golocale "github.com/jeandeaual/go-locale"
	"github.com/leonelquinteros/gotext"
)

//go:generate go run github.com/0skillallluck/scanline/internal/gettext/gen ../../locales/scanline.pot
//go:generate find ../../locales -name "*.po" -exec msgmerge -U -N --backup=off {} ../../locales/scanline.pot ;
var locale *gotext.Locale

func init() {
	userLocale, err := golocale.GetLocale()
	if err != nil {
		slog.Error("could not detect system language, falling back to english")
		userLocale = "en_US"
	}
	locale = gotext.NewLocaleFSWithPath(userLocale, locales.FS, ".")
	locale.AddDomain("default")
}

func Get(msgid string, args ...any) string {

	return locale.Get(msgid, args...)
}

func GetN(msgid string, msgidPlural string, n int, args ...any) string {
	return locale.GetN(msgid, msgidPlural, n, args...)
}
