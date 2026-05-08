package shortcuts

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
	"github.com/0skillallluck/scanline/internal/gettext"
)

// NewShortcutsDialog creates and returns a new keyboard shortcuts dialog.
func NewShortcutsDialog() schwifty.ShortcutsDialog {
	basic := []any{
		ShortcutsItemFromAction(gettext.Get("Close"), "win.close"),
		ShortcutsItemFromAction(gettext.Get("Quit"), "app.quit"),
	}
	if item := mainMenuShortcut(); item != nil {
		basic = append(basic, item)
	}
	basic = append(basic,
		ShortcutsItemFromAction(gettext.Get("Keyboard Shortcuts"), "app.shortcuts"),
		ShortcutsItemFromAction(gettext.Get("Preferences"), "app.preferences"),
	)

	return ShortcutsDialog(
		ShortcutsSection(basic...).Title(gettext.Get("Basic Shortcuts")),
		ShortcutsSection(
			ShortcutsItemFromAction(gettext.Get("Back"), "win.navigate-back"),
			ShortcutsItemFromAction(gettext.Get("Search"), "win.search"),
		).Title(gettext.Get("Navigation")),
	)
}
