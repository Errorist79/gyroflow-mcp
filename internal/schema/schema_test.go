package schema

import (
	"encoding/json"
	"testing"
)

func TestOutParamsRoundTrip(t *testing.T) {
	in := OutParams{Codec: "H.265/HEVC", Bitrate: 150, UseGPU: true, Audio: true}
	b, _ := json.Marshal(in)
	var out OutParams
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: %+v vs %+v", out, in)
	}
}
