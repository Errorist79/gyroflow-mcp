package backend

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGyroflowConfigJSONRoundTrip(t *testing.T) {
	in := `{"smoothing":{"method":"Default","smoothness":0.7},` +
		`"horizon_lock":{"amount":100},"trim":{"ranges_ms":[[0,3000]]},` +
		`"keyframes":[{"param":"Fov","timestamp_ms":1200,"value":1.5,"easing":"EaseIn"}],` +
		`"output":{"codec":"H.265/HEVC","bitrate":150},"raw_overrides":{"x":1}}`
	var c GyroflowConfig
	if err := json.Unmarshal([]byte(in), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Smoothing == nil || c.Smoothing.Method != "Default" || c.Smoothing.Smoothness == nil || *c.Smoothing.Smoothness != 0.7 {
		t.Fatalf("smoothing not parsed: %+v", c.Smoothing)
	}
	if c.HorizonLock == nil || c.HorizonLock.Amount == nil || *c.HorizonLock.Amount != 100 {
		t.Fatalf("horizon_lock not parsed: %+v", c.HorizonLock)
	}
	if len(c.Trim.RangesMS) != 1 || c.Trim.RangesMS[0] != [2]float64{0, 3000} {
		t.Fatalf("trim not parsed: %+v", c.Trim)
	}
	if len(c.Keyframes) != 1 || c.Keyframes[0].Param != "Fov" || c.Keyframes[0].Easing != "EaseIn" {
		t.Fatalf("keyframes not parsed: %+v", c.Keyframes)
	}
	if c.RawOverrides["x"] != float64(1) {
		t.Fatalf("raw_overrides not parsed: %+v", c.RawOverrides)
	}
	// StabilizeRequest carries Config.
	_ = StabilizeRequest{Config: &c}

	// Empty config must marshal to "{}" (omitempty: nothing the user did not set).
	empty, err := json.Marshal(GyroflowConfig{})
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	if string(empty) != "{}" {
		t.Fatalf("empty GyroflowConfig marshaled to %s, want {}", empty)
	}
	// A partial config omits unset top-level keys.
	partial, err := json.Marshal(GyroflowConfig{Smoothing: &SmoothingConfig{Method: "Default"}})
	if err != nil {
		t.Fatalf("marshal partial: %v", err)
	}
	ps := string(partial)
	if !strings.Contains(ps, `"smoothing"`) || strings.Contains(ps, `"horizon_lock"`) || strings.Contains(ps, `"output"`) {
		t.Fatalf("partial config leaked unset keys: %s", ps)
	}
}
