package threadsafe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorBag_SetUnset(t *testing.T) {
	var b ErrorBag

	assert.NoError(t, b.Get())

	err := errors.New("foo")
	b.Set(err)

	assert.EqualError(t, b.Get(), err.Error())

	b.Unset()

	assert.NoError(t, b.Get())
}

func TestErrorBag_Wrap(t *testing.T) {
	var b ErrorBag

	b.Wrap(errors.New("foo"))

	assert.EqualError(t, b.Get(), "foo")

	b.Wrap(errors.New("bar"))

	assert.EqualError(t, b.Get(), "bar: foo")
}
