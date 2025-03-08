package humanize_test

import (
	"math"
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/humanize"
)

func TestPercentageDiff(t *testing.T) {
	t.Parallel()

	assertEqual(t, "0.87%", humanize.PercentageDiff(98.31, 97.46))
	assertEqual(t, "-0.86%", humanize.PercentageDiff(97.46, 98.31))
	assertEqual(t, "2.35%", humanize.PercentageDiff(130.93, 127.92))
	assertEqual(t, "-2.30%", humanize.PercentageDiff(127.92, 130.93))

	assertEqual(t, "0.00%", humanize.PercentageDiff(1, 0))
	assertEqual(t, "-100.00%", humanize.PercentageDiff(0, 1))
	assertEqual(t, "99900.00%", humanize.PercentageDiff(1, 0.001))
	assertEqual(t, "-50.00%", humanize.PercentageDiff(1, 2))
	assertEqual(t, "-90.00%", humanize.PercentageDiff(1, 10))
	assertEqual(t, "-99.00%", humanize.PercentageDiff(1, 100))

	assertEqual(t, "-150.00%", humanize.PercentageDiff(1, -2))
	assertEqual(t, "-110.00%", humanize.PercentageDiff(1, -10))
	assertEqual(t, "-101.00%", humanize.PercentageDiff(1, -100))

	assertEqual(t, "-101.63%", humanize.PercentageDiff(-2, 123))
	assertEqual(t, "-108.13%", humanize.PercentageDiff(-10, 123))
	assertEqual(t, "-181.30%", humanize.PercentageDiff(-100, 123))

	assertEqual(t, "-6250.00%", humanize.PercentageDiff(123, -2))
	assertEqual(t, "-1330.00%", humanize.PercentageDiff(123, -10))
	assertEqual(t, "-223.00%", humanize.PercentageDiff(123, -100))

	assertEqual(t, "-98.37%", humanize.PercentageDiff(-2, -123))
	assertEqual(t, "-91.87%", humanize.PercentageDiff(-10, -123))
	assertEqual(t, "6050.00%", humanize.PercentageDiff(-123, -2))
	assertEqual(t, "1130.00%", humanize.PercentageDiff(-123, -10))

	assertEqual(t, "-Inf%", humanize.PercentageDiff(math.Inf(1), -10))
	assertEqual(t, "+Inf%", humanize.PercentageDiff(math.Inf(-1), -10))

	assertEqual(t, "NaN%", humanize.PercentageDiff(-10, math.Inf(1)))
	assertEqual(t, "NaN%", humanize.PercentageDiff(-10, math.Inf(-1)))
}
