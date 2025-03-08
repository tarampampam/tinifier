package humanize

import (
	"fmt"
)

// PercentageDiff returns a human-readable representation of the percentage difference between two numbers.
func PercentageDiff[A, B number](a A, b B) string {
	var floatA, floatB = float64(a), float64(b)

	if floatB == 0 {
		return "0.00%"
	}

	return fmt.Sprintf("%0.2f%%", ((floatA-floatB)/floatB)*100) //nolint:mnd
}
