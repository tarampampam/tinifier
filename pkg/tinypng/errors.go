package tinypng

import "strings"

// Package-specific errors prefix.
const errorsPrefix = "tinypng.com:"

// Error is a special type for package-specific errors.
type Error uint8

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

// Package-specific error constants.
const (
	ErrTooManyRequests Error = iota + 1 // too many requests (limit has been exceeded)
	ErrUnauthorized                     // unauthorized (invalid credentials)
	ErrBadRequest                       // bad request (empty file or wrong format)
)
