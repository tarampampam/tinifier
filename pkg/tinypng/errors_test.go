package tinypng

import "testing"

func TestConstErr_Error(t *testing.T) {
	cases := []struct {
		name       string
		giveConst  Error
		wantString string
	}{
		{
			name:       "ErrTooManyRequests",
			giveConst:  ErrTooManyRequests,
			wantString: "tinypng.com: too many requests (limit has been exceeded)",
		},
		{
			name:       "ErrUnauthorized",
			giveConst:  ErrUnauthorized,
			wantString: "tinypng.com: unauthorized (invalid credentials)",
		},
		{
			name:       "ErrBadRequest",
			giveConst:  ErrBadRequest,
			wantString: "tinypng.com: bad request (empty file or wrong format)",
		},
		{
			name:       "0",
			giveConst:  Error(0),
			wantString: "tinypng.com: unknown error",
		},
		{
			name:       "255",
			giveConst:  Error(255),
			wantString: "tinypng.com: unknown error",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.giveConst.Error(); tt.wantString != got {
				t.Errorf(`want: "%s", got: "%s"`, tt.wantString, got)
			}
		})
	}
}
