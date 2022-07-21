package logger_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tarampampam/tinifier/v4/internal/logger"
)

func TestAllLevels(t *testing.T) {
	require.EqualValues(t,
		[]logger.Level{logger.DebugLevel, logger.InfoLevel, logger.WarnLevel, logger.ErrorLevel},
		logger.AllLevels(),
	)
}

func TestAllLevelStrings(t *testing.T) {
	require.EqualValues(t, []string{"debug", "info", "warn", "error"}, logger.AllLevelStrings())
}

func TestLevel_String(t *testing.T) {
	for name, tt := range map[string]struct {
		giveLevel  logger.Level
		wantString string
	}{
		"debug":     {giveLevel: logger.DebugLevel, wantString: "debug"},
		"info":      {giveLevel: logger.InfoLevel, wantString: "info"},
		"warn":      {giveLevel: logger.WarnLevel, wantString: "warn"},
		"error":     {giveLevel: logger.ErrorLevel, wantString: "error"},
		"<unknown>": {giveLevel: logger.Level(126), wantString: "level(126)"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.wantString, tt.giveLevel.String())
		})
	}
}

func TestParseLevel(t *testing.T) {
	for name, tt := range map[string]struct {
		giveText  []byte
		wantLevel logger.Level
		wantError error
	}{
		"<empty value>": {giveText: []byte(""), wantLevel: logger.InfoLevel},
		"trace":         {giveText: []byte("debug"), wantLevel: logger.DebugLevel},
		"verbose":       {giveText: []byte("debug"), wantLevel: logger.DebugLevel},
		"debug":         {giveText: []byte("debug"), wantLevel: logger.DebugLevel},
		"info":          {giveText: []byte("info"), wantLevel: logger.InfoLevel},
		"warn":          {giveText: []byte("warn"), wantLevel: logger.WarnLevel},
		"error":         {giveText: []byte("error"), wantLevel: logger.ErrorLevel},
		"foobar":        {giveText: []byte("foobar"), wantError: errors.New("unrecognized logging level: \"foobar\"")},
	} {
		t.Run(name, func(t *testing.T) {
			l, err := logger.ParseLevel(tt.giveText)

			if tt.wantError == nil {
				require.NoError(t, err)
				require.Equal(t, tt.wantLevel, l)
			} else {
				require.EqualError(t, err, tt.wantError.Error())
			}
		})
	}
}
