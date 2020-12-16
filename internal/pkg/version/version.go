package version

import "strings"

// version value will be set during compilation
var version string = "v0.0.0@undefined" // TODO(jetexe) ommit type?

// Version returns version value (without `v` prefix).
func Version() string {
	return strings.TrimLeft(version, "vV ")
}
