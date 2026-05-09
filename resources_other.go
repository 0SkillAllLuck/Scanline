//go:build !darwin

package main

import _ "embed"

//go:embed assets/meta/icons.gresource
var iconBundleLinux []byte

var IconResources = [][]byte{iconBundleLinux}
