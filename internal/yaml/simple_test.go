package yaml_test

import (
	"strings"
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/yaml"
)

func TestDecoder_Decode(t *testing.T) {
	var target struct {
		Int         int      `yaml:"int"`
		String      *string  `yaml:"string"`
		Bool        *bool    `yaml:"bool"`
		Uint        uint     `yaml:"uint"`
		StringSlice []string `yaml:"stringSlice"`
		Struct      struct {
			Int       int     `yaml:"int"`
			String    string  `yaml:"string"`
			StringPtr *string `yaml:"stringPtr"`
		} `yaml:"struct"`
	}

	if err := yaml.NewDecoder(strings.NewReader(`
# comment

int: -12312312342
string: &anchor hello
bool: true
uint: 42
stringSlice: [hello, World]
struct:
  int: !!int 43 # comment
  string: foobar
  stringPtr: *anchor

`)).Decode(&target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, target.Int, -12312312342)
	assertEqual(t, *target.String, "hello")
	assertEqual(t, *target.Bool, true)
	assertEqual(t, target.Uint, uint(42))
	assertEqual(t, len(target.StringSlice), 2)
	assertEqual(t, target.StringSlice[0], "hello")
	assertEqual(t, target.StringSlice[1], "World")
	assertEqual(t, target.Struct.Int, 43)
	assertEqual(t, target.Struct.String, "foobar")
	assertEqual(t, *target.Struct.StringPtr, "hello")
}

// assertEqual checks if two values of a comparable type are equal.
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
