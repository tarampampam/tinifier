package humanize_test

import (
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/humanize"
)

func TestBytes(t *testing.T) {
	t.Parallel()

	assertEqual(t, "-1.00 TB", humanize.Bytes(-1099511627776))
	assertEqual(t, "-500 B", humanize.Bytes(int64(-500)))
	assertEqual(t, "-1 B", humanize.Bytes(int32(-1)))
	assertEqual(t, "0 B", humanize.Bytes(-0))
	assertEqual(t, "0 B", humanize.Bytes(0))
	assertEqual(t, "1 B", humanize.Bytes(uint8(1)))
	assertEqual(t, "500 B", humanize.Bytes(uint(500)))
	assertEqual(t, "512 B", humanize.Bytes(uint32(512)))
	assertEqual(t, "2.00 KB", humanize.Bytes(uint64(2048)))
	assertEqual(t, "1024.00 KB", humanize.Bytes(1048575))
	assertEqual(t, "10.24 KB", humanize.Bytes(10482))
	assertEqual(t, "1.00 MB", humanize.Bytes(1048576))
	assertEqual(t, "5.00 GB", humanize.Bytes(5368709120))
	assertEqual(t, "1.00 TB", humanize.Bytes(1099511627776))
}

func TestBytesDiff(t *testing.T) {
	t.Parallel()

	assertEqual(t, "0 B", humanize.BytesDiff(0, 0))
	assertEqual(t, "1 B", humanize.BytesDiff(uint8(1), 0))
	assertEqual(t, "1 B", humanize.BytesDiff(uint(2), int64(1)))
	assertEqual(t, "-1 B", humanize.BytesDiff(1, 2))
	assertEqual(t, "1 B", humanize.BytesDiff(0, -1))
	assertEqual(t, "1 B", humanize.BytesDiff(-1, -2))
	assertEqual(t, "1 B", humanize.BytesDiff(-1, -2))
	assertEqual(t, "-1019.00 GB", humanize.BytesDiff(5368709120, 1099511627776))
}

// assertEqual checks if two values of a comparable type are equal.
func assertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
