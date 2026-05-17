package lens

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a test helper that writes content to a file in dir.
func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}

// TestRankedSearchRealFixtureBest pins RankedSearch's `best` to the real
// committed GoPro fixture's actual JSON fields (verified against the committed
// fixtures): calib resolution 5312x2988, fps 29.97, the real identifier and
// camera fields.
func TestRankedSearchRealFixtureBest(t *testing.T) {
	idx, err := LoadFromDir("../../testdata/lens/sample_profiles")
	if err != nil {
		t.Fatal(err)
	}
	best, _ := idx.RankedSearch("GoPro HERO11", 5312, 2988, 29.97)
	if best == nil {
		t.Fatal("expected a best match for GoPro HERO11")
	}
	if best.Brand != "GoPro" || best.Model != "HERO11 Black" || best.Lens != "Linear" {
		t.Fatalf("best camera fields wrong: %+v", best)
	}
	if best.Width != 5312 || best.Height != 2988 {
		t.Fatalf("best resolution = %dx%d, want 5312x2988 (real calib_dimension)", best.Width, best.Height)
	}
	if best.FPS < 29.96 || best.FPS > 29.98 {
		t.Fatalf("best fps = %v, want ~29.97 (real .fps)", best.FPS)
	}
	if best.Identifier != "gopro-hero11black-linear-5312x2988@29970-no-eis" {
		t.Fatalf("best identifier = %q, want the real fixture slug", best.Identifier)
	}
}

// TestRankedSearchNearestResolutionFPS verifies the ranking order: among
// same-camera candidates the nearest calib resolution wins, fps breaks ties.
// Uses the real lens-profile JSON schema (calib_dimension/fps pinned in STEP 0).
func TestRankedSearchNearestResolutionFPS(t *testing.T) {
	dir := t.TempDir()
	// Three real-schema profiles for one camera at different modes.
	p := func(id string, w, h int, fps float64) string {
		return fmt.Sprintf(`{"name":"GoPro_HERO11 Black_Linear_16by9","camera_brand":"GoPro",`+
			`"camera_model":"HERO11 Black","lens_model":"Linear","identifier":%q,`+
			`"calib_dimension":{"w":%d,"h":%d},"fps":%g}`, id, w, h, fps)
	}
	if err := writeFile(dir, "a.json", p("gopro-5312-2997", 5312, 2988, 29.97)); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(dir, "b.json", p("gopro-1920-5994", 1920, 1080, 59.94)); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(dir, "c.json", p("gopro-1920-2997", 1920, 1080, 29.97)); err != nil {
		t.Fatal(err)
	}
	idx, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Query a 1920x1080@29.97 clip: nearest res is the two 1920x1080 entries;
	// fps tie-break picks the 29.97 one (c).
	best, alts := idx.RankedSearch("GoPro HERO11", 1920, 1080, 29.97)
	if best == nil || best.Identifier != "gopro-1920-2997" {
		t.Fatalf("best = %+v, want gopro-1920-2997 (nearest res + fps)", best)
	}
	// 5312x2988 entry must be ranked last (largest resolution distance).
	if len(alts) != 2 || alts[len(alts)-1].Identifier != "gopro-5312-2997" {
		t.Fatalf("alternatives mis-ranked: %+v", alts)
	}
}

// TestGyroflowAutoLoadsBoundary pins the embedded-profile family boundary to
// the STEP-0-cited telemetry-parser supported brands.
func TestGyroflowAutoLoadsBoundary(t *testing.T) {
	// Full I-3 matrix (token-anywhere; "red" first-token-only). Every brand in
	// embeddedProfileBrands has a true-case (incl. "xtra", cited in
	// telemetry-parser README as "XTRA (Edge, Edge Pro)"). "HERO11 GoPro" and
	// "stabilize my GoPro footage" prove token-anywhere (not first-token-only).
	for _, q := range []string{
		"GoPro HERO11", "HERO11 GoPro", "stabilize my GoPro footage",
		"DJI Air 2S", "Sony A1", "insta360 X4", "Blackmagic Pocket",
		"Canon C400", "Xtra Edge Pro",
		"RED Komodo", "red komodo", // "red" as the FIRST token → true
	} {
		if !GyroflowAutoLoads(q) {
			t.Errorf("GyroflowAutoLoads(%q) = false, want true", q)
		}
	}
	// Must NOT false-positive: substrings, non-families, and "red" when it is
	// NOT the first token (the ambiguous English-word exception).
	for _, q := range []string{
		"Nikon Z6", "Panasonic GH5", "no such camera xyz",
		"Tokina red ring lens", // "red" not first → false
		"predator cam", "Sonyo XZ", "discanon",
	} {
		if GyroflowAutoLoads(q) {
			t.Errorf("GyroflowAutoLoads(%q) = true, want false", q)
		}
	}
}

func TestSearchByCamera(t *testing.T) {
	idx, err := LoadFromDir("../../testdata/lens/sample_profiles")
	if err != nil {
		t.Fatal(err)
	}
	hits := idx.Search("GoPro HERO11")
	if len(hits) == 0 {
		t.Fatal("expected at least one match for GoPro HERO11")
	}
	if hits[0].Path == "" {
		t.Fatal("hit missing path")
	}
}

// TestSearchCaseInsensitive verifies the query is matched regardless of case.
func TestSearchCaseInsensitive(t *testing.T) {
	idx, err := LoadFromDir("../../testdata/lens/sample_profiles")
	if err != nil {
		t.Fatal(err)
	}
	hits := idx.Search("gopro hero11")
	if len(hits) == 0 {
		t.Fatal("expected case-insensitive match for 'gopro hero11'")
	}
}

// TestSearchNoMatch verifies an unrecognised query returns empty results.
func TestSearchNoMatch(t *testing.T) {
	idx, err := LoadFromDir("../../testdata/lens/sample_profiles")
	if err != nil {
		t.Fatal(err)
	}
	hits := idx.Search("no such camera brand xyz123")
	if len(hits) != 0 {
		t.Fatalf("expected no hits, got %d", len(hits))
	}
}

// TestLoadFromDirSkipsMalformed verifies that a directory with a bad JSON file
// still loads the valid profiles (best-effort, no abort).
func TestLoadFromDirSkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	// Write a valid profile.
	valid := `{"name":"Test","camera_brand":"Acme","camera_model":"X1","lens_model":"","identifier":"acme-x1"}`
	if err := writeFile(dir, "valid.json", valid); err != nil {
		t.Fatal(err)
	}
	// Write an invalid JSON file.
	if err := writeFile(dir, "bad.json", `{not valid json`); err != nil {
		t.Fatal(err)
	}
	idx, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir should not fail on malformed file: %v", err)
	}
	if len(idx.hits) != 1 {
		t.Fatalf("expected 1 valid hit, got %d", len(idx.hits))
	}
}

// TestHitDisplayContainsModel verifies the Display field is built from real fields.
func TestHitDisplayContainsModel(t *testing.T) {
	idx, err := LoadFromDir("../../testdata/lens/sample_profiles")
	if err != nil {
		t.Fatal(err)
	}
	hits := idx.Search("GoPro HERO11")
	if len(hits) == 0 {
		t.Fatal("expected match")
	}
	if hits[0].Display == "" {
		t.Fatal("Display must not be empty")
	}
}

// TestExtractTarGzRejectsPathTraversal builds an in-memory .tar.gz containing a
// benign profile and a malicious "../../../evil.json" entry, extracts it into a
// temp dir, and asserts the benign file is written under destDir while the evil
// path never escapes destDir (zip-slip / tar path traversal prevention).
func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	write := func(name, body string) {
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(body)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	// Benign entry (top-level prefix is stripped by extractTarGz).
	write("lens_profiles-main/good.json", `{"name":"Good","camera_brand":"Acme","identifier":"acme-good"}`)
	// Malicious entry attempting to escape destDir.
	write("lens_profiles-main/../../../evil.json", `{"name":"Evil"}`)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	if err := extractTarGz(&buf, destDir); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	// Benign file must exist under destDir.
	if _, err := os.Stat(filepath.Join(destDir, "good.json")); err != nil {
		t.Fatalf("benign good.json not written under destDir: %v", err)
	}
	// The evil file must NOT exist anywhere outside destDir.
	parent := filepath.Dir(destDir)
	for _, p := range []string{
		filepath.Join(parent, "evil.json"),
		filepath.Join(filepath.Dir(parent), "evil.json"),
		filepath.Join(destDir, "..", "..", "..", "evil.json"),
	} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("path traversal escaped destDir: %s exists", p)
		}
	}
}

// TestLoadFromDirSkipsEmptyProfile verifies a profile with BOTH empty name and
// empty identifier (e.g. "{}") is not indexed (no all-spaces junk hit).
func TestLoadFromDirSkipsEmptyProfile(t *testing.T) {
	dir := t.TempDir()
	if err := writeFile(dir, "real.json",
		`{"name":"Real","camera_brand":"Acme","camera_model":"Z","identifier":"acme-z"}`); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(dir, "empty.json", `{}`); err != nil {
		t.Fatal(err)
	}
	idx, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.Len(); got != 1 {
		t.Fatalf("expected 1 indexed profile (empty {} skipped), got %d", got)
	}
	// The skipped junk file's filename-stem id must not resolve.
	if _, ok := idx.ByID("empty"); ok {
		t.Fatal("skipped empty profile must not be resolvable by id")
	}
}

// TestByIDSemantics pins ByID behavior (identical before and after the O(1)
// map refactor): known identifier, filename-stem fallback, unknown, and
// first-wins on duplicate ids.
func TestByIDSemantics(t *testing.T) {
	idx, err := LoadFromDir("../../testdata/lens/sample_profiles")
	if err != nil {
		t.Fatal(err)
	}
	// Known identifier-field id (GoPro fixture).
	h, ok := idx.ByID("gopro-hero11black-linear-5312x2988@29970-no-eis")
	if !ok || !strings.HasSuffix(h.Path, "GoPro_HERO11_Black_Linear_16by9.json") {
		t.Fatalf("known identifier lookup failed: ok=%v hit=%+v", ok, h)
	}
	// Filename-stem fallback id (DJI fixture has empty identifier).
	if _, ok := idx.ByID("DJI_AIR_2S_4k_29.97fps"); !ok {
		t.Fatal("filename-stem fallback id should resolve")
	}
	// Unknown id.
	if _, ok := idx.ByID("no-such-id-xyz"); ok {
		t.Fatal("unknown id must not resolve")
	}

	// First-wins on duplicate ids.
	dir := t.TempDir()
	if err := writeFile(dir, "a.json",
		`{"name":"A","camera_brand":"Acme","camera_model":"A","identifier":"dup-id"}`); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(dir, "b.json",
		`{"name":"B","camera_brand":"Acme","camera_model":"B","identifier":"dup-id"}`); err != nil {
		t.Fatal(err)
	}
	didx, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	hit, ok := didx.ByID("dup-id")
	if !ok {
		t.Fatal("duplicate id should resolve to first match")
	}
	// LoadFromDir walks lexically; a.json is walked before b.json → first wins.
	if !strings.HasSuffix(hit.Path, "a.json") {
		t.Fatalf("first-wins violated: expected a.json, got %s", hit.Path)
	}
}
