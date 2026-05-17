package mcpsurface

import (
	"strings"
	"testing"
)

// TestCapabilitiesResource verifies capabilitiesDoc returns a non-empty Markdown
// reference containing every documented semantic/enum/range token an LLM needs
// to translate natural-language requests into typed render_start config values.
func TestCapabilitiesResource(t *testing.T) {
	body := capabilitiesDoc()
	for _, want := range []string{
		"adaptive_zoom_window", "-1", "horizon_lock.amount", "0..100",
		"frame_readout_direction", "TopToBottom", "KeyframeType",
		"gopro_superview", "gyroflow/gyroflow@v1.6.3",
		"typed path wins on conflict",
		"To MINIMISE cropping use -1", // directive: minimal-crop guidance
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("capabilities doc missing %q", want)
		}
	}
}
