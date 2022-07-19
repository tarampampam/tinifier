package keys_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/tinifier/v4/internal/keys"
)

func TestKeeper_Add_Get_Report(t *testing.T) {
	k := keys.NewKeeper()

	// get on empty state must returns an error
	_, err := k.Get()
	assert.EqualError(t, err, keys.ErrKeyNotExists.Error())

	// appending without error
	assert.NoError(t, k.Add("foo", "bar"))

	// even two keys in a state - first by default will be returned
	got, err := k.Get()
	assert.NoError(t, err)
	assert.Contains(t, []string{"foo", "bar"}, got)

	// remove first key
	k.Remove("foo")

	for i := 0; i < 100; i++ {
		got, err = k.Get()
		assert.NoError(t, err)
		assert.Equal(t, "bar", got)
	}

	// remove second key
	k.Remove("bar")
	_, err = k.Get()
	assert.EqualError(t, err, keys.ErrKeyNotExists.Error())
}

func TestKeeper_Add(t *testing.T) {
	k := keys.NewKeeper()

	assert.Error(t, k.Add(""))               // empty key
	assert.Error(t, k.Add("foo", "", "bar")) // empty key

	assert.NoError(t, k.Add("bar"))

	assert.Error(t, k.Add("foo")) // duplicate
}

func TestKeeper_Remove(t *testing.T) {
	k := keys.NewKeeper()

	assert.NoError(t, k.Add("foo", "bar"))

	k.Remove("foo")
	k.Remove() // for coverage only

	got, err := k.Get()
	assert.NoError(t, err)
	assert.Equal(t, "bar", got)

	k.Remove("bar", "some", "another", "keys")

	_, err = k.Get()
	assert.EqualError(t, err, keys.ErrKeyNotExists.Error())
}
