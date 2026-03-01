package about

import (
	"fmt"
	"regexp"

	"github.com/0skillallluck/scanline/internal/gettext"
	"github.com/jwijenbergh/puregotk/v4/adw"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

var (
	Commit, Version string
)

// IsStable returns true if the current build is a stable release.
func IsStable() bool {
	if Version != "" {
		return true
	}

	if Commit == "" {
		// If no commit is available.
		// This happens if the app is run with `go run main.go`.
		return false
	} else if ok, _ := regexp.MatchString(`^.*-\d+-g[0-9a-f]{7}$`, Commit); ok {
		// If the commit matches the git describe scheme.
		// This happens when the code is run locally, through a project-provided tool, and a tag is available.
		regex := regexp.MustCompile(`^(.*)-(\d+)-g([0-9a-f]{7})`)
		parts := regex.FindStringSubmatch(Commit)

		offset := parts[2]

		return offset == "0"
	} else {
		// If the commit is not a git describe scheme.
		// This happens when the code is run locally, through a project-provided tool, and no tag is available.
		return false
	}
}

func getVersionPrefix(version string) (prefix string) {
	if version[0] == 'v' {
		if version[1] == '0' {
			prefix = "α "
		}
	} else {
		if version[0] == '0' {
			prefix = "α "
		} else {
			prefix = "v"
		}
	}

	return
}

func getVersionNumber() string {
	if Version == "" {
		// If no version is available.
		// This happens if the go binary is run directly, or through the Zed task

		if Commit == "" {
			// If no commit is available.
			// This happens if the app is run with `go run main.go`.
			return "local"
		} else if ok, _ := regexp.MatchString(`^.*-\d+-g[0-9a-f]{7}$`, Commit); ok {
			// If the commit matches the git describe scheme.
			// This happens when the code is run locally, through a project-provided tool, and a tag is available.
			// We can therefore display a git version, in the `v1.0.0 β(r1.0123abc)` scheme.
			regex := regexp.MustCompile(`^(.*)-(\d+)-g([0-9a-f]{7})`)
			parts := regex.FindStringSubmatch(Commit)

			_, tag, offset, commitHash := parts[0], parts[1], parts[2], parts[3]
			if tag[0] == 'v' {
				tag = tag[1:]
			}

			prefix := getVersionPrefix(tag)

			suffix := ""
			if offset != "0" {
				suffix = "β"
			}

			return fmt.Sprintf("%s%s %s(r%s.%s)", prefix, tag, suffix, offset, commitHash)
		} else {
			// If the commit is not a git describe scheme.
			// This happens when the code is run locally, through a project-provided tool, and no tag is available.
			// We can therefore display a commit hash, with the alpha symbol.
			return fmt.Sprintf("α git+%s", Commit[:7])
		}
	} else {
		// If a version has been provided.
		// We can therefore display the version, and the commit hash for clarity.
		commit := Commit
		suffix := ""
		if commit != "" {
			suffix = fmt.Sprintf(" (%s)", commit[:7])
		}

		version := Version
		if version[0] == 'v' {
			version = version[1:]
		}

		prefix := getVersionPrefix(version)

		return fmt.Sprintf("%s%s%s", prefix, version, suffix)
	}
}

// NewAboutDialog creates and returns a new about dialog for the application.
func NewAboutDialog() *adw.AboutDialog {
	dialog := adw.NewAboutDialog()
	dialog.SetApplicationIcon("logo")
	dialog.SetApplicationName("Scanline")
	dialog.SetVersion(getVersionNumber())
	dialog.SetLicenseType(gtk.LicenseGpl30Value)
	dialog.SetDevelopers([]string{
		"Nila The Dragon https://github.com/NilaTheDragon",
		"Dråfølin https://github.com/Drafolin",
	})
	dialog.SetTranslatorCredits(gettext.Get("translator-credits"))
	dialog.SetCopyright("© 2026 Nila The Dragon")
	dialog.SetWebsite("https://scanline.skillless.dev")
	dialog.SetIssueUrl("https://github.com/0skillallluck/scanline/issues")
	dialog.SetSupportUrl("https://matrix.to/#/%23scanline:skillless.dev")

	dialog.AddLegalSection("GStreamer Bindings (go-gst/go-gst)", "© 2020 https://github.com/go-gst/go-gst", gtk.LicenseLgpl30Value, "")
	dialog.AddLegalSection("DBus Client (godbus/dbus)", "© 2020 Georg Reinke https://github.com/godbus/dbus", gtk.LicenseBsdValue, "")
	dialog.AddLegalSection("UUID (google/uuid)", "© 2009, 2014 Google Inc. https://github.com/google/uuid", gtk.LicenseBsd3Value, "")
	dialog.AddLegalSection("System Locale Detection (jeandeaual/go-locale)", "© 2020 Alexis Jeandeau https://github.com/jeandeaual/go-locale", gtk.LicenseMitX11Value, "")
	dialog.AddLegalSection("GTK4 / Libadwaita Bindings (jwijenbergh/puregotk)", "© 2022 Kyle McGough https://github.com/jwijenbergh/puregotk", gtk.LicenseMitX11Value, "")
	dialog.AddLegalSection("Libsecret (lescuer97/go-libsecret)", "© 2025 Leonardo Escuer https://github.com/lescuer97/go-libsecret", gtk.LicenseMitX11Value, "")
	dialog.AddLegalSection("JSON Merger (qjebbs/go-jsons)", "© 2022 Jebbs https://github.com/qjebbs/go-jsons", gtk.LicenseMitX11Value, "")
	dialog.AddLegalSection("ISO8601 Duration Parser (osodev/duration)", "© 2023 Jeroen Wijenbergh https://github.com/sosodev/duration", gtk.LicenseMitX11Value, "")
	dialog.AddLegalSection("QR Code Generator (yeqown/go-qrcode)", "© 2018 yeqown https://github.com/yeqown/go-qrcode", gtk.LicenseMitX11Value, "")
	dialog.AddLegalSection("Gettext(leonelquinteros/gotext)", "© 2016 Leonel Quinteros https://github.com/leonelquinteros/gotext", gtk.LicenseMitX11Value, "")

	return dialog
}
