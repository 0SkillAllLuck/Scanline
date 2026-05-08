//go:build darwin

package shortcuts

import adwbindings "codeberg.org/dergs/tonearm/pkg/schwifty/bindings/adw"

// mainMenuShortcut returns nil on macOS — the in-app hamburger menu (and its
// F10 shortcut) doesn't exist there; the menu lives in NSMainMenu instead.
func mainMenuShortcut() adwbindings.ShortcutsItem { return nil }
