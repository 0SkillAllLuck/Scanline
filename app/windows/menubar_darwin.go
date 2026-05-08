//go:build darwin

package windows

import (
	adwbindings "codeberg.org/dergs/tonearm/pkg/schwifty/bindings/adw"
	"codeberg.org/puregotk/puregotk/v4/gio"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"github.com/0skillallluck/scanline/internal/gettext"
)

// installNativeMenubar wires the application's menu model into NSMainMenu via
// gtk_application_set_menubar. GTK4's macOS backend auto-generates the
// application menu (About / Settings / Hide / Quit) from the standard Cocoa
// items, so we only contribute File / Edit / Help here — adding a "Scanline"
// submenu would render as a duplicate after the auto-generated app menu.
func (w *Window) installNativeMenubar() {
	menubar := gio.NewMenu()

	fileMenu := gio.NewMenu()
	fileMenu.Append(gettext.Get("Select Sources…"), "win.select-sources")
	appendSection(fileMenu, gettext.Get("Close Window"), "win.close")
	menubar.AppendSubmenu(gettext.Get("File"), &fileMenu.MenuModel)

	editMenu := gio.NewMenu()
	editMenu.Append(gettext.Get("Find"), "win.search")
	menubar.AppendSubmenu(gettext.Get("Edit"), &editMenu.MenuModel)

	windowMenu := gio.NewMenu()
	windowMenu.Append(gettext.Get("Minimize"), "win.minimize")
	windowMenu.Append(gettext.Get("Zoom"), "win.zoom")
	appendSection(windowMenu, gettext.Get("Bring All to Front"), "app.bring-all-to-front")
	appendSection(windowMenu, gettext.Get("Close Window"), "win.close")
	// gtk-macos-special=window-submenu opts the submenu into AppKit's native
	// Window menu management (open-window list, standard window items).
	windowItem := gio.NewMenuItemSubmenu(gettext.Get("Window"), &windowMenu.MenuModel)
	windowItem.SetAttributeValue("gtk-macos-special", glib.NewVariantString("window-submenu"))
	menubar.AppendItem(windowItem)

	helpMenu := gio.NewMenu()
	helpMenu.Append(gettext.Get("About Scanline"), "app.about")
	helpMenu.Append(gettext.Get("Keyboard Shortcuts"), "app.shortcuts")
	menubar.AppendSubmenu(gettext.Get("Help"), &helpMenu.MenuModel)

	w.GetApplication().SetMenubar(&menubar.MenuModel)
}

func (w *Window) packMainMenuButton(b adwbindings.HeaderBar) adwbindings.HeaderBar {
	return b
}

func appendSection(parent *gio.Menu, label, action string) {
	section := gio.NewMenu()
	section.Append(label, action)
	parent.AppendSection("", &section.MenuModel)
}
