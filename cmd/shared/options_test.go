package shared

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIKey_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "foo", (WithAPIKey{APIKey: "foo"}).APIKey.String())
}

func TestAPIKey_Masked(t *testing.T) {
	t.Parallel()

	tests := []struct{ give, want string }{
		{give: "foobarbarfoo", want: "foob****rfoo"},
		{give: "foobar", want: "foobar"},
		{give: "foo1234567890bar", want: "foo1********0bar"},
		{give: "", want: ""},
		{give: "********", want: "********"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s->%s", tt.give, tt.want), func(t *testing.T) {
			assert.Equal(t, tt.want, (WithAPIKey{APIKey: APIKey(tt.give)}).APIKey.Masked())
		})
	}
}
