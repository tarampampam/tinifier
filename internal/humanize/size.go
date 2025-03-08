package humanize

import "fmt"

// Bytes returns a human-readable representation of a size in bytes.
func Bytes[T integer](bytes T) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	var sign, asInt = "", int64(bytes)

	if bytes < 0 {
		sign, bytes = "-", -bytes
	}

	switch {
	case asInt >= tb:
		return fmt.Sprintf("%s%.2f TB", sign, float64(bytes)/float64(tb))
	case asInt >= gb:
		return fmt.Sprintf("%s%.2f GB", sign, float64(bytes)/float64(gb))
	case asInt >= mb:
		return fmt.Sprintf("%s%.2f MB", sign, float64(bytes)/float64(mb))
	case asInt >= kb:
		return fmt.Sprintf("%s%.2f KB", sign, float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%s%d B", sign, bytes)
	}
}

// BytesDiff returns a human-readable representation of the difference between two sizes in bytes.
func BytesDiff[A, B integer](a A, b B) string {
	return Bytes(int64(a) - int64(b))
}
