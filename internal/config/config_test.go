package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/config"
)

func TestConfig_FromFile(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		giveContent   string
		wantStruct    config.Config
		wantErrSubstr string
	}{
		"empty file": {
			giveContent: "",
			wantStruct:  config.Config{},
		},
		"full config": {
			giveContent: `
apiKeys: [foo, bar, baz]`,
			wantStruct: func() (c config.Config) {
				c.ApiKeys = toPtr([]string{"foo", "bar", "baz"})

				return
			}(),
		},

		"broken yaml": {
			giveContent:   "$rossia-budet-svobodnoy$",
			wantErrSubstr: "failed to decode the config file",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var filePath = filepath.Join(t.TempDir(), "config.yml")

			if err := os.WriteFile(filePath, []byte(tc.giveContent), 0o600); err != nil {
				t.Fatalf("failed to create a config file: %v", err)
			}

			var (
				c   config.Config
				err = c.FromFile(filePath)
			)

			if tc.wantErrSubstr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// assert the structure
				if !reflect.DeepEqual(c, tc.wantStruct) {
					t.Fatalf("expected: %+v, got: %+v", tc.wantStruct, c)
				}

				return
			}

			if err == nil {
				t.Fatalf("expected an error, got nil")
			}

			if got := err.Error(); !strings.Contains(got, tc.wantErrSubstr) {
				t.Fatalf("expected error to contain %q, got %q", tc.wantErrSubstr, got)
			}
		})
	}

	t.Run("merge", func(t *testing.T) {
		var (
			tmpDir  = t.TempDir()
			config1 = filepath.Join(tmpDir, "config1.yml")
			config2 = filepath.Join(tmpDir, "config2.yml")
		)

		// create config files
		for _, err := range []error{
			os.WriteFile(config1, []byte(`
apiKeys: [foo, bar]
`), 0o600),
			os.WriteFile(config2, []byte(`
apiKeys: [bar, baz]
`), 0o600),
		} {
			if err != nil {
				t.Fatalf("failed to create a config file: %v", err)
			}
		}

		var cfg config.Config

		// read the first file
		if err := cfg.FromFile(config1); err != nil {
			t.Fatalf("failed to read the first config file: %v", err)
		}

		// read the second file
		if err := cfg.FromFile(config2); err != nil {
			t.Fatalf("failed to read the second config file: %v", err)
		}

		// assert the structure
		if !reflect.DeepEqual(cfg, config.Config{
			ApiKeys: toPtr([]string{"bar", "baz"}),
		}) {
			t.Fatalf("unexpected config: %+v", cfg)
		}
	})
}

func toPtr[T any](v T) *T { return &v }
