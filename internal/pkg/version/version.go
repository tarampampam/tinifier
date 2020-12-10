package version

import "strings"

// version value will be set during compilation
var version string = "v0.0.0"

// Version returns version value (without `v` prefix).
func Version() string {
	return strings.TrimLeft(version, "vV ")
}
