package tinypng_test

import (
	"testing"

	"gh.tarampamp.am/tinifier/v5/pkg/tinypng"
)

func TestClientsPool_Get(t *testing.T) {
	t.Parallel()

	var pool = tinypng.NewClientsPool([]string{"foo", "bar"})

	client1, rm1, found1 := pool.Get() // first client

	if client1 == nil {
		t.Error("client is nil")
	}

	if rm1 == nil {
		t.Error("removing function is nil")
	}

	if !found1 {
		t.Error("client not found")
	}

	rm1() // remove client from pool

	client2, rm2, found2 := pool.Get() // second client

	if client2 == nil {
		t.Error("client is nil")
	}

	if rm2 == nil {
		t.Error("removing function is nil")
	}

	if !found2 {
		t.Error("client not found")
	}

	if client1 == client2 {
		t.Error("clients are the same")
	}

	rm2() // remove client from pool

	noClient, noRm, notFound := pool.Get() // 3rd attempt

	if noClient != nil {
		t.Error("expected nil client")
	}

	if noRm == nil {
		t.Error("removing function is nil")
	}

	if notFound {
		t.Error("client found, but expected not found")
	}

	noRm() // noop
}
