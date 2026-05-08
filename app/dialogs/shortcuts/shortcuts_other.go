//go:build !darwin

package shortcuts

import (
	adwbindings "codeberg.org/dergs/tonearm/pkg/schwifty/bindings/adw"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
)

func mainMenuShortcut() adwbindings.ShortcutsItem {
	return ShortcutsItemFromAction(gettext.Get("Main Menu"), "win.main-menu")
}
