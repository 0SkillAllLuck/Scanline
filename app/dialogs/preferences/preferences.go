package preferences

import (
	"codeberg.org/dergs/tonearm/pkg/schwifty/bindings/adw"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
)

func NewPreferencesDialog() adw.PreferencesDialog {
	return PreferencesDialog(
		generalPreferences,
		playerPreferences,
		performancePreferences,
		experimentalPreferences,
	)
}
