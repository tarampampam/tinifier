package logger_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tarampampam/tinifier/v4/internal/logger"
)

func TestAllFormats(t *testing.T) {
	require.EqualValues(t, []logger.Format{logger.ConsoleFormat, logger.JSONFormat}, logger.AllFormats())
}

func TestAllFormatStrings(t *testing.T) {
	require.EqualValues(t, []string{"console", "json"}, logger.AllFormatStrings())
}

func TestFormat_String(t *testing.T) {
	for name, tt := range map[string]struct {
		giveFormat logger.Format
		wantString string
	}{
		"json":      {giveFormat: logger.JSONFormat, wantString: "json"},
		"console":   {giveFormat: logger.ConsoleFormat, wantString: "console"},
		"<unknown>": {giveFormat: logger.Format(255), wantString: "format(255)"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.wantString, tt.giveFormat.String())
		})
	}
}

func TestParseFormat(t *testing.T) {
	for name, tt := range map[string]struct {
		giveText   []byte
		wantFormat logger.Format
		wantError  error
	}{
		"<empty value>": {giveText: []byte(""), wantFormat: logger.ConsoleFormat},
		"console":       {giveText: []byte("console"), wantFormat: logger.ConsoleFormat},
		"json":          {giveText: []byte("json"), wantFormat: logger.JSONFormat},
		"foobar":        {giveText: []byte("foobar"), wantError: errors.New("unrecognized logging format: \"foobar\"")},
	} {
		t.Run(name, func(t *testing.T) {
			f, err := logger.ParseFormat(tt.giveText)

			if tt.wantError == nil {
				require.NoError(t, err)
				require.Equal(t, tt.wantFormat, f)
			} else {
				require.EqualError(t, err, tt.wantError.Error())
			}
		})
	}
}
