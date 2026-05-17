// Package gyroflow locates and describes the installed Gyroflow CLI.
package gyroflow

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// ErrNotFound means no usable gyroflow binary was located.
var ErrNotFound = errors.New("gyroflow binary not found")

// The real `gyroflow --version` prints multi-line CombinedOutput: Qt DEBUG
// lines, a "gyroflow: Gyroflow X.Y.Z" debug line, and a clean "Gyroflow vX.Y.Z"
// line. This regex matches the version on ANY of those lines (case-insensitive,
// optional 'v'), so detection survives Gyroflow changing or dropping debug output.
// Verified against gyroflow 1.6.3 - see testdata/version_output.txt.
var versionRe = regexp.MustCompile(`(?i)gyroflow\s+v?(\d+\.\d+\.\d+)`)

func parseVersion(out string) (string, error) {
	m := versionRe.FindStringSubmatch(out)
	if m == nil {
		return "", errors.New("could not parse gyroflow version from output")
	}
	return m[1], nil
}

// Info describes a detected binary.
type Info struct {
	Path    string
	Version string
}

// candidatePaths returns likely binary locations per OS, in priority order.
func candidatePaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Applications/Gyroflow.app/Contents/MacOS/gyroflow"}
	case "windows":
		return []string{`C:\Program Files\Gyroflow\gyroflow.exe`}
	default:
		return []string{"/usr/local/bin/gyroflow", "/opt/gyroflow/gyroflow"}
	}
}

// Detect finds the binary on PATH first, then OS-standard locations,
// and confirms it by running `--version`.
func Detect(ctx context.Context) (*Info, error) {
	var tryPaths []string
	if p, err := exec.LookPath("gyroflow"); err == nil {
		tryPaths = append(tryPaths, p)
	}
	tryPaths = append(tryPaths, candidatePaths()...)

	for _, p := range tryPaths {
		out, err := exec.CommandContext(ctx, p, "--version").CombinedOutput()
		if err != nil {
			continue
		}
		v, err := parseVersion(strings.TrimSpace(string(out)))
		if err != nil {
			continue
		}
		return &Info{Path: p, Version: v}, nil
	}
	return nil, ErrNotFound
}
