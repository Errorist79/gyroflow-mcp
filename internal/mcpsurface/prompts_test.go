package mcpsurface

import (
	"strings"
	"testing"
)

func TestStabilizePromptTeachesConfigFlow(t *testing.T) {
	text := stabilizeFootagePromptText("<the input video path>")
	for _, want := range []string{
		"gyroflow://capabilities",
		"horizon_lock.amount", // unique to the NL→config intent-mapping section
		"ONE render_start call",
		"do NOT",    // existing no-external-tools steering retained
		"crop MORE", // directive: positive adaptive_zoom_window crops MORE not less
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}
