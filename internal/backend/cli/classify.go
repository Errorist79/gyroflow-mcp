package cli

import "strings"

// classifyRunFailure inspects combined gyroflow output for known failure patterns
// and returns actionable advice. matched is false when no known pattern is found.
//
// Matched on substring "Unable to read the video file", e.g.:
//
//	"...Unable to read the video file." / "[add_file]: Unable to read the video file."
func classifyRunFailure(combinedOutput, goos string) (advice string, matched bool) {
	if strings.Contains(combinedOutput, "Unable to read the video file") {
		if goos == "darwin" {
			return "gyroflow could not read the input. Most common cause on macOS: the installed Gyroflow is the sandboxed App Store build (run gyroflow_doctor; install the DMG build via `brew install --cask gyroflow`). Also ensure the path is absolute and the file exists.", true
		}
		return "gyroflow could not read the input. Ensure the path is absolute and the file exists.", true
	}
	return "", false
}
