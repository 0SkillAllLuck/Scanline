//go:build darwin

package accels

// GTK4 on macOS labels <Primary> as ⌘ in shortcut dialogs but only matches
// Control at match-time; the Cmd key arrives as <Meta>. Bind <Meta> directly
// so the accelerator both displays and fires correctly on Cmd.
const PrimaryMod = "<Meta>"
