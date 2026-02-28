package search

import (
	"github.com/0skillallluck/scanline/internal/gettext"
	"codeberg.org/dergs/tonearm/pkg/schwifty"
	. "codeberg.org/dergs/tonearm/pkg/schwifty/syntax"
)

func PromptView() schwifty.StatusPage {
	return StatusPage().
		IconName("loupe-symbolic").
		Title(gettext.Get("Search")).
		Description(gettext.Get("Start typing in the search bar to search for movies, shows, and more."))
}

func LoadingView() schwifty.Clamp {
	return Clamp().
		MaximumSize(50).
		Child(Spinner())
}

func NoResultsView() schwifty.StatusPage {
	return StatusPage().
		IconName("loupe-symbolic").
		Title(gettext.Get("No Results Found")).
		Description(gettext.Get("Try a different search term."))
}
