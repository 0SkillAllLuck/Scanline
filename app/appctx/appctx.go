package appctx

import (
	"context"

	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/0skillallluck/scanline/app/sources"
)

type AppContext struct {
	Ctx     context.Context
	Cancel  context.CancelFunc
	Manager *sources.Manager
	Window  *gtk.Window
}
