package appctx

import (
	"context"

	"github.com/0skillallluck/scanline/app/sources"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

type AppContext struct {
	Ctx     context.Context
	Cancel  context.CancelFunc
	Manager *sources.Manager
	Window  *gtk.Window
}
