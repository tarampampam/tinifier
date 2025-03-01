package cmd_test

import (
	"errors"
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/cli/cmd"
)

func TestFlag_IsSet(t *testing.T) {
	t.Parallel()

	assertEqual(t, (&cmd.Flag[string]{
		// no value
	}).IsSet(), false, "uninitialized flag should not be set")

	assertEqual(t, (&cmd.Flag[int]{
		Value:        new(int),
		ValueSetFrom: cmd.FlagValueSourceNone,
	}).IsSet(), false, "flag with no value should not be set")

	assertEqual(t, (&cmd.Flag[int]{
		Value:        new(int),
		ValueSetFrom: cmd.FlagValueSourceDefault,
	}).IsSet(), false, "flag with default value should not be set")

	var intValue = 42

	assertEqual(t, (&cmd.Flag[int]{
		Value:        &intValue,
		ValueSetFrom: cmd.FlagValueSourceFlag,
		Default:      intValue,
	}).IsSet(), false, "flag with value that equals to default should not be set")

	assertEqual(t, (&cmd.Flag[bool]{
		Value:        new(bool),
		ValueSetFrom: cmd.FlagValueSourceFlag,
		Default:      true,
	}).IsSet(), true, "flag with value that differs from default should be set")
}

func TestFlag_Help(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		giveFlag             cmd.Flag[string]
		wantNames, wantUsage string
	}{
		"empty": {
			giveFlag:  cmd.Flag[string]{},
			wantNames: "",
			wantUsage: "",
		},
		"single long name": {
			giveFlag:  cmd.Flag[string]{Names: []string{"name"}},
			wantNames: `--name="…"`,
			wantUsage: "",
		},
		"single short name": {
			giveFlag:  cmd.Flag[string]{Names: []string{"n"}},
			wantNames: `-n="…"`,
			wantUsage: "",
		},
		"multiple names": {
			giveFlag:  cmd.Flag[string]{Names: []string{"name", "n"}},
			wantNames: `--name="…", -n="…"`,
			wantUsage: "",
		},
		"with usage": {
			giveFlag:  cmd.Flag[string]{Usage: "usage\nfoo"},
			wantNames: "",
			wantUsage: "usage\nfoo",
		},
		"with default": {
			giveFlag:  cmd.Flag[string]{Default: "default"},
			wantNames: "",
			wantUsage: "(default: default)",
		},
		"with env vars": {
			giveFlag:  cmd.Flag[string]{EnvVars: []string{"ENV1", "ENV2"}},
			wantNames: "",
			wantUsage: "[$ENV1, $ENV2]",
		},
		"full": {
			giveFlag: cmd.Flag[string]{
				Names:   []string{"name", "n"},
				Usage:   "usage\nfoo",
				Default: "default",
				EnvVars: []string{"ENV1", "ENV2"},
			},
			wantNames: `--name="…", -n="…"`,
			wantUsage: "usage\nfoo (default: default) [$ENV1, $ENV2]",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotNames, gotUsage := tc.giveFlag.Help()

			assertEqual(t, gotNames, tc.wantNames, "unexpected names")
			assertEqual(t, gotUsage, tc.wantUsage, "unexpected usage")
		})
	}

	t.Run("bool", func(t *testing.T) {
		t.Run("default true", func(t *testing.T) {
			gotNames, gotUsage := (&cmd.Flag[bool]{
				Names:   []string{"name", "n"},
				Usage:   "usage\nfoo",
				Default: true,
				EnvVars: []string{"ENV1", "ENV2"},
			}).Help()

			assertEqual(t, gotNames, `--name, -n`, "unexpected names")
			assertEqual(t, gotUsage, "usage\nfoo (default: true) [$ENV1, $ENV2]", "unexpected usage")
		})

		t.Run("default false", func(t *testing.T) {
			gotNames, gotUsage := (&cmd.Flag[bool]{
				Names:   []string{"name"},
				Usage:   "usage\nfoo",
				Default: false,
				EnvVars: []string{"ENV1", "ENV2"},
			}).Help()

			assertEqual(t, gotNames, `--name`, "unexpected names")
			assertEqual(t, gotUsage, "usage\nfoo [$ENV1, $ENV2]", "unexpected usage")
		})
	})
}

func TestFlag_Apply(t *testing.T) {
	t.Parallel()

	t.Run("bool, default", func(t *testing.T) {
		t.Parallel()

		var (
			val bool
			f   = &cmd.Flag[bool]{Names: []string{"test"}, Value: &val, Default: true}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, true, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("bool, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val bool
			f   = &cmd.Flag[bool]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test"}))
		assertEqual(t, val, true, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("bool, env", func(t *testing.T) {
		t.Parallel()

		var (
			envName = setRandomEnv(t, "True")
			val     bool
			f       = &cmd.Flag[bool]{Names: []string{"test"}, Value: &val, EnvVars: []string{envName}}
			set     = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, true, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceEnv, "unexpected value source")
	})

	t.Run("bool, wrong env", func(t *testing.T) {
		t.Parallel()

		var (
			envName = setRandomEnv(t, "<invalid+Boolean-Value]")
			val     bool
			f       = &cmd.Flag[bool]{Names: []string{"test"}, EnvVars: []string{envName}}
			set     = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, false, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("int, default", func(t *testing.T) {
		t.Parallel()

		var (
			val int
			f   = &cmd.Flag[int]{Names: []string{"test"}, Value: &val, Default: 42}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, 42, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("int, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val int
			f   = &cmd.Flag[int]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test=42"}))
		assertEqual(t, val, 42, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("int, wrong flag", func(t *testing.T) {
		t.Parallel()

		var (
			val int
			f   = &cmd.Flag[int]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.ContinueOnError)
		)

		f.Apply(set)

		assertErrorContains(t, set.Parse([]string{"--test=foo"}), "must contain only digits with an optional leading")
		assertEqual(t, val, 0, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("int, env", func(t *testing.T) {
		t.Parallel()

		var (
			envName = setRandomEnv(t, "42")
			val     int
			f       = &cmd.Flag[int]{Names: []string{"test"}, Value: &val, EnvVars: []string{envName}}
			set     = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, 42, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceEnv, "unexpected value source")
	})

	t.Run("int, wrong env", func(t *testing.T) {
		t.Parallel()

		var (
			envName = setRandomEnv(t, "forty-two")
			val     int
			f       = &cmd.Flag[int]{Names: []string{"test"}, EnvVars: []string{envName}}
			set     = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, 0, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("int64, default", func(t *testing.T) {
		t.Parallel()

		var (
			val int64
			f   = &cmd.Flag[int64]{Names: []string{"test"}, Value: &val, Default: 42}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, int64(42), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("int64, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val int64
			f   = &cmd.Flag[int64]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test=-42"}))
		assertEqual(t, val, int64(-42), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("int64, wrong flag", func(t *testing.T) {
		t.Parallel()

		var (
			val int64
			f   = &cmd.Flag[int64]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.ContinueOnError)
		)

		f.Apply(set)

		assertErrorContains(t, set.Parse([]string{"--test=foo"}), "must contain only digits with an optional leading")
		assertEqual(t, val, int64(0), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("string, default", func(t *testing.T) {
		t.Parallel()

		var (
			val string
			f   = &cmd.Flag[string]{Names: []string{"test"}, Value: &val, Default: "default"}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, "default", "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("string, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val string
			f   = &cmd.Flag[string]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test=foo"}))
		assertEqual(t, val, "foo", "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("string, env", func(t *testing.T) {
		t.Parallel()

		var (
			envVal  = randomString(10)
			envName = setRandomEnv(t, envVal)

			val string
			f   = &cmd.Flag[string]{Names: []string{"test"}, Value: &val, EnvVars: []string{envName}}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, envVal, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceEnv, "unexpected value source")
	})

	t.Run("uint, default", func(t *testing.T) {
		t.Parallel()

		var (
			val uint
			f   = &cmd.Flag[uint]{Names: []string{"test"}, Value: &val, Default: 42}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, uint(42), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("uint, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val uint
			f   = &cmd.Flag[uint]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test=42"}))
		assertEqual(t, val, uint(42), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("uint, wrong flag", func(t *testing.T) {
		t.Parallel()

		var (
			val uint
			f   = &cmd.Flag[uint]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.ContinueOnError)
		)

		f.Apply(set)

		assertErrorContains(t, set.Parse([]string{"--test=foo"}), "must contain only digits")
		assertEqual(t, val, uint(0), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("uint64, default", func(t *testing.T) {
		t.Parallel()

		var (
			val uint64
			f   = &cmd.Flag[uint64]{Names: []string{"test"}, Value: &val, Default: 42}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, uint64(42), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("uint64, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val uint64
			f   = &cmd.Flag[uint64]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test=42"}))
		assertEqual(t, val, uint64(42), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("uint64, wrong flag", func(t *testing.T) {
		t.Parallel()

		var (
			val uint64
			f   = &cmd.Flag[uint64]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.ContinueOnError)
		)

		f.Apply(set)

		assertErrorContains(t, set.Parse([]string{"--test=foo"}), "must contain only digits")
		assertEqual(t, val, uint64(0), "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("float64, default", func(t *testing.T) {
		t.Parallel()

		var (
			val float64
			f   = &cmd.Flag[float64]{Names: []string{"test"}, Value: &val, Default: 42.42}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, 42.42, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("float64, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val float64
			f   = &cmd.Flag[float64]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test=42.42"}))
		assertEqual(t, val, 42.42, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("float64, wrong flag", func(t *testing.T) {
		t.Parallel()

		var (
			val float64
			f   = &cmd.Flag[float64]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.ContinueOnError)
		)

		f.Apply(set)

		assertErrorContains(t, set.Parse([]string{"--test=foo"}), "must contain only digits with an optional decimal")
		assertEqual(t, val, 0.0, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("time.Duration, default", func(t *testing.T) {
		t.Parallel()

		var (
			val time.Duration
			f   = &cmd.Flag[time.Duration]{Names: []string{"test"}, Value: &val, Default: time.Second}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, time.Second, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("time.Duration, flag", func(t *testing.T) {
		t.Parallel()

		var (
			val time.Duration
			f   = &cmd.Flag[time.Duration]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse([]string{"--test=42s"}))
		assertEqual(t, val, 42*time.Second, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceFlag, "unexpected value source")
	})

	t.Run("time.Duration, wrong flag", func(t *testing.T) {
		t.Parallel()

		var (
			val time.Duration
			f   = &cmd.Flag[time.Duration]{Names: []string{"test"}, Value: &val}
			set = newFlagSet(flag.ContinueOnError)
		)

		f.Apply(set)

		assertErrorContains(t, set.Parse([]string{"--test=foo"}), "must be a valid Go duration string")
		assertEqual(t, val, 0*time.Second, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})

	t.Run("time.Duration, env", func(t *testing.T) {
		t.Parallel()

		var (
			envVal  = "42s"
			envName = setRandomEnv(t, envVal)

			val time.Duration
			f   = &cmd.Flag[time.Duration]{Names: []string{"test"}, Value: &val, EnvVars: []string{envName}}
			set = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, 42*time.Second, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceEnv, "unexpected value source")
	})

	t.Run("time.Duration, wrong env", func(t *testing.T) {
		t.Parallel()

		var (
			envName = setRandomEnv(t, "forty-two")
			val     time.Duration
			f       = &cmd.Flag[time.Duration]{Names: []string{"test"}, EnvVars: []string{envName}}
			set     = newFlagSet(flag.PanicOnError)
		)

		f.Apply(set)

		assertNoError(t, set.Parse(nil))
		assertEqual(t, val, 0*time.Second, "unexpected value")
		assertEqual(t, f.ValueSetFrom, cmd.FlagValueSourceDefault, "unexpected value source")
	})
}

func TestFlag_Validate(t *testing.T) {
	t.Parallel()

	t.Run("nil validator", func(t *testing.T) {
		t.Parallel()

		assertNoError(t, (&cmd.Flag[string]{Names: []string{"test"}}).Validate(&cmd.Command{}))
	})

	t.Run("nil value", func(t *testing.T) {
		t.Parallel()

		var (
			executed bool

			f = &cmd.Flag[string]{
				Names: []string{"test"},
				Validator: func(command *cmd.Command, s string) error {
					executed = true

					return nil
				},
			}
		)

		assertErrorContains(t, f.Validate(&cmd.Command{}), "flag value is nil")
		assertEqual(t, executed, false, "unexpected execution")
	})

	t.Run("validator error", func(t *testing.T) {
		t.Parallel()

		var (
			executed bool
			val      = "str"

			f = &cmd.Flag[string]{
				Names: []string{"test"},
				Value: &val,
				Validator: func(command *cmd.Command, s string) error {
					assertEqual(t, s, val, "unexpected value")

					executed = true

					return errors.New("test error")
				},
			}
		)

		assertErrorContains(t, f.Validate(&cmd.Command{}), "test error")
		assertEqual(t, executed, true, "should be executed")
	})
}

func TestFlag_RunAction(t *testing.T) {
	t.Parallel()

	t.Run("nil action", func(t *testing.T) {
		t.Parallel()

		assertNoError(t, (&cmd.Flag[string]{Names: []string{"test"}}).RunAction(&cmd.Command{}))
	})

	t.Run("nil value", func(t *testing.T) {
		t.Parallel()

		var (
			executed bool

			f = &cmd.Flag[string]{
				Names: []string{"test"},
				Action: func(command *cmd.Command, s string) error {
					executed = true

					return nil
				},
			}
		)

		assertErrorContains(t, f.RunAction(&cmd.Command{}), "flag value is nil")
		assertEqual(t, executed, false, "unexpected execution")
	})

	t.Run("action error", func(t *testing.T) {
		t.Parallel()

		var (
			executed bool
			val      = "str"

			f = &cmd.Flag[string]{
				Names: []string{"test"},
				Value: &val,
				Action: func(command *cmd.Command, s string) error {
					assertEqual(t, s, val, "unexpected value")

					executed = true

					return errors.New("test error")
				},
			}
		)

		assertErrorContains(t, f.RunAction(&cmd.Command{}), "test error")
		assertEqual(t, executed, true, "should be executed")
	})
}

func newFlagSet(eh flag.ErrorHandling) *flag.FlagSet {
	var set = flag.NewFlagSet("test", eh)

	set.SetOutput(io.Discard)

	return set
}

func setRandomEnv(t *testing.T, value string) (envName string) {
	t.Helper()

	envName = randomString(10)

	assertNoError(t, os.Setenv(envName, value))

	t.Cleanup(func() { assertNoError(t, os.Unsetenv(envName)) })

	return envName
}
