package version

import "testing"

func TestVersion(t *testing.T) {
	t.Parallel()

	if value := Version(); value != "0.0.0@undefined" {
		t.Errorf("Unexpected default version value: %s", value)
	}
}
