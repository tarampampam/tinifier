package tinypng

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Error is a special type for package-specific errors.
type Error uint8

// Package-specific error constants.
const (
	ErrTooManyRequests Error = iota + 1 // too many requests (limit has been exceeded)
	ErrUnauthorized                     // unauthorized (invalid credentials)
	ErrBadRequest                       // bad request (empty file or wrong format)
)

// Package-specific errors prefix.
const errorsPrefix = "tinypng.com:"

// Error returns error in a string representation.
func (err Error) Error() string {
	var buf strings.Builder
	defer buf.Reset() // GC is our bro

	buf.WriteString(errorsPrefix + " ")

	switch err {
	case ErrTooManyRequests:
		buf.WriteString("too many requests (limit has been exceeded)")

	case ErrUnauthorized:
		buf.WriteString("unauthorized (invalid credentials)")

	case ErrBadRequest:
		buf.WriteString("bad request (empty file or wrong format)")

	default:
		buf.WriteString("unknown error")
	}

	return buf.String()
}

func newError(message string) error {
	return fmt.Errorf(errorsPrefix + " " + message)
}

func newErrorf(format string, args ...any) error {
	return errors.New(errorsPrefix + " " + fmt.Errorf(format, args...).Error())
}

// parseRemoteError reads HTTP response content as a JSON-string, parse them and converts into go-error.
//
// This function should never return nil!
func parseRemoteError(content io.Reader) error {
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(content).Decode(&e); err != nil {
		return newErrorf("error decoding failed: %w", err)
	}

	return newErrorf("%s (%s)", e.Error, strings.Trim(e.Message, ". "))
}
