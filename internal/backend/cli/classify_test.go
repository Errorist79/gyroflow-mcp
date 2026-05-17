package cli

import "testing"

// Real captured error strings from gyroflow 1.6.3 (App Store sandboxed build).
// Pinned so the test breaks if pattern matching regresses.
const (
	realReadErrFull = `23:04:26 [ERROR] [7ffe6b56] Error: An error occured: Unable to read the video file.`
	realReadErrWarn = `[WARN] [add_file]: Unable to read the video file.`
)

func TestClassifyRunFailureReadErrorDarwin(t *testing.T) {
	advice, matched := classifyRunFailure(realReadErrFull, "darwin")
	if !matched {
		t.Fatal("expected matched=true for read error on darwin")
	}
	if advice == "" {
		t.Fatal("expected non-empty advice for darwin read error")
	}
}

func TestClassifyRunFailureReadErrorWarnDarwin(t *testing.T) {
	advice, matched := classifyRunFailure(realReadErrWarn, "darwin")
	if !matched {
		t.Fatal("expected matched=true for WARN read error on darwin")
	}
	if advice == "" {
		t.Fatal("expected non-empty advice for darwin WARN read error")
	}
}

func TestClassifyRunFailureReadErrorLinux(t *testing.T) {
	advice, matched := classifyRunFailure(realReadErrFull, "linux")
	if !matched {
		t.Fatal("expected matched=true for read error on linux")
	}
	if advice == "" {
		t.Fatal("expected non-empty advice for linux read error")
	}
}

func TestClassifyRunFailureNoMatch(t *testing.T) {
	_, matched := classifyRunFailure("some unrelated gyroflow error", "darwin")
	if matched {
		t.Fatal("expected matched=false for unrelated error")
	}
}
