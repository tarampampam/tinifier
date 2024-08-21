package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstants(t *testing.T) {
	require.Equal(t, "FORCE_COLOR", string(ForceColors))
	require.Equal(t, "NO_COLOR", string(NoColors))
	require.Equal(t, "TERM", string(Term))
	require.Equal(t, "THREADS_COUNT", string(ThreadsCount))
	require.Equal(t, "TINYPNG_API_KEY", string(TinyPngAPIKey))
}

func TestEnvVariable_Lookup(t *testing.T) {
	cases := []struct {
		giveEnv envVariable
	}{
		{giveEnv: TinyPngAPIKey},
	}

	for _, tt := range cases {
		t.Run(tt.giveEnv.String(), func(t *testing.T) {
			require.NoError(t, os.Unsetenv(tt.giveEnv.String())) // make sure that env is unset for test

			defer func() { require.NoError(t, os.Unsetenv(tt.giveEnv.String())) }()

			value, exists := tt.giveEnv.Lookup()
			assert.False(t, exists)
			assert.Empty(t, value)

			assert.NoError(t, os.Setenv(tt.giveEnv.String(), "foo"))

			value, exists = tt.giveEnv.Lookup()
			assert.True(t, exists)
			assert.Equal(t, "foo", value)
		})
	}
}
