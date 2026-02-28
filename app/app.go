package app

import (
	"context"
	"log/slog"

	"github.com/0skillallluck/scanline/app/appctx"
	"github.com/0skillallluck/scanline/app/dialogs/secret_service"
	"github.com/0skillallluck/scanline/app/router"
	"github.com/0skillallluck/scanline/app/sources"
	"github.com/0skillallluck/scanline/app/windows"
	"github.com/0skillallluck/scanline/app/secrets"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gio"
)

func OnActivate(application *adw.Application) func(gio.Application) {
	return func(_ gio.Application) {
		ctx, cancel := context.WithCancel(context.Background())
		mgr := sources.NewManager()

		appCtx := &appctx.AppContext{
			Ctx:     ctx,
			Cancel:  cancel,
			Manager: mgr,
		}

		// Cancel context and wait for in-flight operations on shutdown
		application.ConnectShutdown(new(func(gio.Application) {
			cancel()
			router.Wait()
		}))

		window := windows.NewWindow(application, appCtx)
		appCtx.Window = &window.Window

		window.Present()

		if err := secrets.Healthcheck(); err != nil {
			slog.Error("Secret service health check failed", "title", err.Title, "body", err.Body, "fatal", err.Fatal)
			secret_service.PresentSecretServiceErrorDialog(err, &window.Widget)
		}
	}
}
