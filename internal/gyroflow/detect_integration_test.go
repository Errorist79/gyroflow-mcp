//go:build integration

package gyroflow

import (
	"context"
	"testing"
)

func TestDetectRealBinary(t *testing.T) {
	info, err := Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v (install Gyroflow to run integration tests)", err)
	}
	if info.Version == "" || info.Path == "" {
		t.Fatalf("incomplete info: %+v", info)
	}
	t.Logf("detected gyroflow %s at %s", info.Version, info.Path)
}
