package cmd_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/cli/cmd"
)

func TestCommand_Help(t *testing.T) {
	t.Parallel()

	var builtInFlagsHelp = `Options:
   --help, -h     Show help
   --version, -v  Print the version`

	for name, tc := range map[string]struct {
		giveCommand *cmd.Command
		wantHelp    string
	}{
		"empty": {
			giveCommand: &cmd.Command{},
			wantHelp:    builtInFlagsHelp,
		},
		"with description": {
			giveCommand: &cmd.Command{
				Description: "Some description here",
			},
			wantHelp: "Description:\n   Some description here\n\n" + builtInFlagsHelp,
		},
		"with name": {
			giveCommand: &cmd.Command{
				Name: "some-name",
			},
			wantHelp: "Usage:\n   some-name\n\n" + builtInFlagsHelp,
		},
		"with name and usage": {
			giveCommand: &cmd.Command{
				Name:  "some-name",
				Usage: "some-usage",
			},
			wantHelp: "Usage:\n   some-name some-usage\n\n" + builtInFlagsHelp,
		},
		"full": {
			giveCommand: &cmd.Command{
				Name:        "some-name",
				Description: "Some description here",
				Usage:       "some-usage",
				Version:     "some-version",
				Flags: []cmd.Flagger{
					&cmd.Flag[string]{
						Names:   []string{"config-file", "c"},
						Usage:   "Path to the configuration file",
						EnvVars: []string{"CONFIG_FILE"},
					},
				},
			},
			wantHelp: `Description:
   Some description here

Usage:
   some-name some-usage

Version:
   some-version

Options:
   --config-file="…", -c="…"  Path to the configuration file [$CONFIG_FILE]
   --help, -h                 Show help
   --version, -v              Print the version`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assertEqual(t, tc.giveCommand.Help(), tc.wantHelp)
		})
	}
}

func TestCommand_Run(t *testing.T) {
	t.Parallel()

	var ctx = context.Background()

	t.Run("cancelled context", func(t *testing.T) {
		t.Parallel()

		var c = &cmd.Command{}

		newCtx, cancel := context.WithCancel(ctx)
		cancel()

		assertErrorContains(t, c.Run(newCtx, nil), context.Canceled.Error())
	})

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		var c = &cmd.Command{}

		assertNoError(t, c.Run(ctx, nil))
		assertNoError(t, c.Run(nil, nil)) //nolint:contextcheck,staticcheck
	})

	t.Run("help (built-in flag)", func(t *testing.T) {
		t.Parallel()

		var (
			out      strings.Builder
			executed bool

			c = &cmd.Command{
				Name:   "some-name",
				Output: &out,
				Action: func(context.Context, *cmd.Command, []string) (_ error) { executed = true; return },
			}
		)

		for _, arg := range [...]string{"--help", "-h"} {
			assertNoError(t, c.Run(ctx, []string{arg}))
			assertEqual(t, executed, false) // should not execute the action
			assertEqual(t, out.String(), c.Help()+"\n")

			out.Reset()
		}
	})

	t.Run("version (built-in flag)", func(t *testing.T) {
		t.Parallel()

		var (
			out            strings.Builder
			runtimeVersion = runtime.Version()
			executed       bool

			c = &cmd.Command{
				Name:    "some-name",
				Version: "some-version",
				Output:  &out,
				Action:  func(context.Context, *cmd.Command, []string) (_ error) { executed = true; return },
			}
		)

		for _, arg := range [...]string{"--version", "-v"} {
			assertNoError(t, c.Run(ctx, []string{arg}))
			assertEqual(t, executed, false) // should not execute the action
			assertEqual(t, out.String(), fmt.Sprintf("%s (%s)\n", c.Version, runtimeVersion))

			out.Reset()
		}

		c.Version = "" // unset version

		for _, arg := range [...]string{"--version", "-v"} {
			assertNoError(t, c.Run(ctx, []string{arg}))
			assertEqual(t, executed, false) // should not execute the action
			assertEqual(t, out.String(), fmt.Sprintf("unknown (%s)\n", runtimeVersion))

			out.Reset()
		}
	})

	t.Run("custom flag action", func(t *testing.T) {
		t.Parallel()

		var (
			cmdActExecuted  bool
			flagActExecuted bool
			testErr         = errors.New("test error")

			c = &cmd.Command{
				Name: "some-name",
				Flags: []cmd.Flagger{
					&cmd.Flag[bool]{
						Names:  []string{"custom-flag", "f"},
						Action: func(_ *cmd.Command, _ bool) error { flagActExecuted = true; return testErr },
					},
				},
				Action: func(context.Context, *cmd.Command, []string) error { cmdActExecuted = true; return nil },
			}
		)

		for _, arg := range [...]string{"--custom-flag", "-f"} {
			assertErrorContains(t, c.Run(ctx, []string{arg}), testErr.Error())
			assertEqual(t, flagActExecuted, true)
			assertEqual(t, cmdActExecuted, false)

			cmdActExecuted, flagActExecuted = false, false // reset
		}
	})

	t.Run("custom flag validation", func(t *testing.T) {
		t.Parallel()

		var (
			value   string
			testErr = errors.New("invalid value")

			c = &cmd.Command{
				Name: "some-name",
				Flags: []cmd.Flagger{
					&cmd.Flag[string]{
						Names: []string{"custom-flag", "f"},
						Validator: func(_ *cmd.Command, s string) error {
							if s == "valid" {
								return nil
							}

							return testErr
						},
						Value: &value,
					},
				},
			}
		)

		// valid value
		for _, args := range [...][]string{
			{"--custom-flag=valid"},
			{"--custom-flag", "valid"},
			{"-f=valid"},
			{"-f", "valid"},
		} {
			assertNoError(t, c.Run(ctx, args))
			assertEqual(t, value, "valid")

			value = "" // reset
		}

		// invalid value
		for _, args := range [...][]string{
			{"--custom-flag=invalid"},
			{"--custom-flag", "invalid"},
			{"-f=invalid"},
			{"-f", "invalid"},
		} {
			assertEqual(t, c.Run(ctx, args), testErr)
			assertEqual(t, value, "invalid") // the value is set anyway

			value = "" // reset
		}
	})

	t.Run("custom flag parsing error", func(t *testing.T) {
		t.Parallel()

		var (
			value int
			out   strings.Builder

			c = &cmd.Command{
				Name:   "some-name",
				Output: &out,
				Flags: []cmd.Flagger{
					&cmd.Flag[bool]{
						Names: []string{"bar"},
					},
					&cmd.Flag[int]{
						Names: []string{"custom-flag", "f"},
						Value: &value,
					},
				},
			}
		)

		for _, args := range [...][]string{
			{"--bar", "--custom-flag=foo"},
			{"--custom-flag", "foo"},
			{"-f=foo"},
			{"-f", "foo", "--bar"},
		} {
			assertErrorContains(t, c.Run(ctx, args), "invalid value")
			assertEqual(t, out.String(), c.Help()+"\n")

			out.Reset() // reset
		}
	})

	t.Run("command action", func(t *testing.T) {
		t.Parallel()

		var (
			executed bool

			c = &cmd.Command{
				Name:   "some-name",
				Action: func(context.Context, *cmd.Command, []string) error { executed = true; return nil },
			}
		)

		assertNoError(t, c.Run(ctx, nil))
		assertEqual(t, executed, true)
	})
}
