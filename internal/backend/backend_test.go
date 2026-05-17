package backend

import "testing"

func TestStabilizeRequestZeroValue(t *testing.T) {
	var r StabilizeRequest
	if r.Inputs != nil {
		t.Fatal("expected nil Inputs on zero value")
	}
}
