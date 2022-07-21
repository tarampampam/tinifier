package logger

import (
	"bytes"
	"fmt"
	"math"
)

// A Level is a logging level.
type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel        // default level (zero-value)
	WarnLevel
	ErrorLevel

	noLevel Level = math.MaxInt8 // no level
)

// AllLevels returns all logging levels.
func AllLevels() []Level { return []Level{DebugLevel, InfoLevel, WarnLevel, ErrorLevel} }

// AllLevelStrings returns all logging levels as a strings slice.
func AllLevelStrings() []string {
	var (
		levels = AllLevels()
		result = make([]string, len(levels))
	)

	for i := 0; i < len(levels); i++ {
		result[i] = levels[i].String()
	}

	return result
}

// String returns a lower-case ASCII representation of the log level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case noLevel:
		return "none"
	}

	return fmt.Sprintf("level(%d)", l)
}

// ParseLevel parses a level (case is ignored) based on the ASCII representation of the log level.
// If the provided ASCII representation is invalid an error is returned.
//
// This is particularly useful when dealing with text input to configure log levels.
func ParseLevel(text []byte) (Level, error) {
	switch string(bytes.ToLower(text)) {
	case "debug", "verbose", "trace":
		return DebugLevel, nil
	case "info", "": // make the zero value useful
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	}

	return Level(0), fmt.Errorf("unrecognized logging level: %q", text)
}
