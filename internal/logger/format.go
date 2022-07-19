package logger

import (
	"bytes"
	"fmt"
)

// A Format is a logging format.
type Format uint8

const (
	ConsoleFormat Format = iota // useful for console output (for humans)
	JSONFormat                  // useful for logging aggregation systems (for robots)
)

// AllFormats returns all logging formats.
func AllFormats() []Format { return []Format{ConsoleFormat, JSONFormat} }

// AllFormatStrings returns all logging formats as a strings slice.
func AllFormatStrings() []string {
	var (
		formats = AllFormats()
		result  = make([]string, len(formats))
	)

	for i := 0; i < len(formats); i++ {
		result[i] = formats[i].String()
	}

	return result
}

// String returns a lower-case ASCII representation of the log format.
func (f Format) String() string {
	switch f {
	case ConsoleFormat:
		return "console"
	case JSONFormat:
		return "json"
	}

	return fmt.Sprintf("format(%d)", f)
}

// ParseFormat parses a format (case is ignored) based on the ASCII representation of the log format.
// If the provided ASCII representation is invalid an error is returned.
//
// This is particularly useful when dealing with text input to configure log formats.
func ParseFormat(text []byte) (Format, error) {
	switch string(bytes.ToLower(text)) {
	case "console", "": // make the zero value useful
		return ConsoleFormat, nil
	case "json":
		return JSONFormat, nil
	}

	return Format(0), fmt.Errorf("unrecognized logging format: %q", text)
}
