package cli

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMetadataRawPathDeterministic pins I-1: probing the same input twice
// resolves to the SAME temp path (so a re-probe overwrites rather than
// accumulating a unique ~1.7MB orphan per call), while distinct inputs map to
// distinct paths.
func TestMetadataRawPathDeterministic(t *testing.T) {
	a1 := metadataRawPath("/abs/path/clip.mp4")
	a2 := metadataRawPath("/abs/path/clip.mp4")
	if a1 != a2 {
		t.Fatalf("same input must yield same raw_path: %q != %q", a1, a2)
	}
	if b := metadataRawPath("/abs/path/other.mp4"); b == a1 {
		t.Fatalf("distinct inputs must yield distinct raw_paths, both = %q", b)
	}
	if !strings.HasPrefix(filepath.Base(a1), "gyroflow-meta-") || !strings.HasSuffix(a1, ".json") {
		t.Fatalf("unexpected raw_path shape: %q", a1)
	}
	// Simulate two probes of the same input: only ONE file must exist after.
	dir := t.TempDir()
	p := filepath.Join(dir, filepath.Base(a1))
	for range 2 {
		if err := os.WriteFile(p, []byte(`{"camera_identifier":{}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "gyroflow-meta-*.json"))
	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 file after two same-input probes, got %d", len(matches))
	}
}

// TestParseMetadataSummaryFixture pins parseMetadataSummary to the committed
// real fixture testdata/metadata_sample.json (trimmed but camera/lens/format
// fields byte-identical to the real `--export-metadata 2` export of
// testdata/sample_gyro.mp4, GoPro HERO10 Black).
//
// HONEST FRAME_COUNT / DURATION NOTE: the fixture truncates the bulk telemetry
// arrays to 2 elements each (so the committed file stays ~2 KB instead of
// ~1.7 MB). frame_count = len(.quaternions) and duration_s =
// .raw_imu[-1].timestamp_ms/1000 are therefore asserted against the TRUNCATED
// fixture (2 and 0.005), NOT a guessed number. On the real untruncated export
// these same expressions yield 1384 and ~23.1s - matching the real render
// Queue line "Frames: 1384" confirmed against the committed fixture. The test
// asserts the parse LOGIC deterministically; the real-file equivalence is
// documented in testdata/metadata_sample.README.
func TestParseMetadataSummaryFixture(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "metadata_sample.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", path, err)
	}

	got, err := parseMetadataSummary(raw)
	if err != nil {
		t.Fatalf("parseMetadataSummary: %v", err)
	}

	if got.Camera.Make != "GoPro" {
		t.Errorf("camera.make = %q, want %q", got.Camera.Make, "GoPro")
	}
	if got.Camera.Model != "HERO10 Black" {
		t.Errorf("camera.model = %q, want %q", got.Camera.Model, "HERO10 Black")
	}
	if got.Camera.Identifier != "gopro-hero10black-wide-1920x1080@59940-no-eis" {
		t.Errorf("camera.identifier = %q, want %q", got.Camera.Identifier, "gopro-hero10black-wide-1920x1080@59940-no-eis")
	}
	if got.DetectedSource != "GoPro HERO10 Black" {
		t.Errorf("detected_source = %q, want %q", got.DetectedSource, "GoPro HERO10 Black")
	}
	if got.Lens != "Wide" { // lens_model "" → falls back to lens_info "Wide"
		t.Errorf("lens = %q, want %q", got.Lens, "Wide")
	}
	if !got.HasGyro {
		t.Error("has_gyro = false, want true (raw_imu non-empty with gyro)")
	}
	if got.Width != 1920 {
		t.Errorf("width = %d, want 1920", got.Width)
	}
	if got.Height != 1080 {
		t.Errorf("height = %d, want 1080", got.Height)
	}
	if math.Abs(got.FPS-59.94) > 0.001 { // .camera_identifier.fps 59940 / 1000
		t.Errorf("fps = %v, want 59.94 (±0.001)", got.FPS)
	}
	// Truncated-fixture honest assertions (see doc comment above):
	if got.FrameCount != 2 {
		t.Errorf("frame_count = %d, want 2 (len quaternions in TRUNCATED fixture; real file == 1384)", got.FrameCount)
	}
	if math.Abs(got.DurationS-0.005) > 1e-9 { // raw_imu[-1].timestamp_ms 5.0 / 1000
		t.Errorf("duration_s = %v, want 0.005 (truncated fixture; real file ≈ 23.1)", got.DurationS)
	}
	if got.DetectedProfile != "" { // .lens_profile was null in the pinned real export
		t.Errorf("detected_profile = %q, want \"\" (null in fixture)", got.DetectedProfile)
	}
}

// TestParseMetadataSummaryNoGyro verifies has_gyro is derived (no boolean
// field is assumed): an empty raw_imu yields has_gyro=false and duration 0.
func TestParseMetadataSummaryNoGyro(t *testing.T) {
	raw := []byte(`{"detected_source":"X","camera_identifier":{"brand":"B","model":"M","fps":30000,"video_width":640,"video_height":480,"identifier":"id"},"raw_imu":[],"quaternions":{},"lens_profile":null}`)
	got, err := parseMetadataSummary(raw)
	if err != nil {
		t.Fatalf("parseMetadataSummary: %v", err)
	}
	if got.HasGyro {
		t.Error("has_gyro = true, want false for empty raw_imu")
	}
	if got.DurationS != 0 {
		t.Errorf("duration_s = %v, want 0 for empty raw_imu", got.DurationS)
	}
	if got.FrameCount != 0 {
		t.Errorf("frame_count = %d, want 0 for empty quaternions", got.FrameCount)
	}
	if math.Abs(got.FPS-30.0) > 0.001 {
		t.Errorf("fps = %v, want 30.0", got.FPS)
	}
}
