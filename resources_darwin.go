//go:build darwin

package main

import _ "embed"

//go:embed assets/meta/icons.gresource
var iconBundleLinux []byte

//go:embed assets/meta/icons-darwin.gresource
var iconBundleDarwin []byte

// IconResources lists icon GResource bundles to register, in order. gio
// prepends to its registry, so later entries win icon-name lookups. The
// Linux bundle stays registered on Darwin so direct resource-path lookups
// (missing-album.svg, rating-source logos) still resolve.
var IconResources = [][]byte{iconBundleLinux, iconBundleDarwin}
