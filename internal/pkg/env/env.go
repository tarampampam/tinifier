// Package env contains all about environment variables, that can be used by current application.
package env

import "os"

type envVariable string

const (
	TinyPngAPIKey envVariable = "TINYPNG_API_KEY" // TinyPNG API key
)

// String returns environment variable name in the string representation.
func (e envVariable) String() string { return string(e) }

// Lookup retrieves the value of the environment variable. If the variable is present in the environment the value
// (which may be empty) is returned and the boolean is true. Otherwise the returned value will be empty and the
// boolean will be false.
func (e envVariable) Lookup() (string, bool) { return os.LookupEnv(string(e)) }
