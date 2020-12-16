package keys

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeeper_Add_Get_Report(t *testing.T) {
	k := NewKeeper(5)

	// get on empty state must returns an error
	_, err := k.Get()
	assert.EqualError(t, err, ErrNoUsableKey.Error())

	// appending without error
	assert.NoError(t, k.Add("foo", "bar"))

	// even two keys in a state - first by default will be returned
	got, err := k.Get()
	assert.NoError(t, err)
	assert.Contains(t, []string{"foo", "bar"}, got)

	// report first key
	assert.NoError(t, k.ReportKey("foo", 4))

	assert.Error(t, k.ReportKey("non-exists", 0)) // non-exists key trigger an error

	// logic without changes
	got, err = k.Get()
	assert.NoError(t, err)
	assert.Contains(t, []string{"foo", "bar"}, got)

	// one more report - and first key is no longer used
	assert.NoError(t, k.ReportKey("foo", 1))

	for i := 0; i < 100; i++ {
		got, err = k.Get()
		assert.NoError(t, err)
		assert.Equal(t, "bar", got)
	}

	// report second key
	assert.NoError(t, k.ReportKey("bar", 999))
	_, err = k.Get()
	assert.EqualError(t, err, ErrNoUsableKey.Error())
}

func TestKeeper_Add(t *testing.T) {
	k := NewKeeper(5)

	assert.Error(t, k.Add(""))               // empty key
	assert.Error(t, k.Add("foo", "", "bar")) // empty key

	assert.NoError(t, k.Add("bar"))

	assert.Error(t, k.Add("foo")) // duplicate
}

func TestKeeper_Remove(t *testing.T) {
	k := NewKeeper(5)

	assert.NoError(t, k.Add("foo", "bar"))

	k.Remove("foo")
	k.Remove() // for coverage only

	got, err := k.Get()
	assert.NoError(t, err)
	assert.Equal(t, "bar", got)

	k.Remove("bar", "some", "another", "keys")

	_, err = k.Get()
	assert.EqualError(t, err, ErrNoUsableKey.Error())
}
