package preferences

import (
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
)

var playerPreferences = PreferencesPage().Title(gettext.Get("Player")).IconName("play")
