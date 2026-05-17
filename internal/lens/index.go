// Package lens provides a local search index for Gyroflow lens profiles
// downloaded from github.com/gyroflow/lens_profiles.
package lens

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// maxLensFileBytes caps a single extracted profile file as defense-in-depth
// against a malicious archive entry claiming an enormous size. Real profiles
// are tens of KB; 64 MiB is a generous ceiling.
const maxLensFileBytes = 64 << 20

// profileJSON mirrors the fields needed for search from the real lens profile
// JSON schema used by github.com/gyroflow/lens_profiles.
//
// Real field names verified by inspecting profiles fetched on 2026-05-17:
//
//	name            - human-readable profile name (e.g. "GoPro_HERO11 Black_Linear_16by9")
//	camera_brand    - manufacturer (e.g. "GoPro", "Sony", "DJI")
//	camera_model    - model string (e.g. "HERO11 Black", "A1", "AIR 2S")
//	lens_model      - lens description; may be empty (e.g. "Linear", "Sigma 14-24mm", "")
//	identifier      - slug used internally (e.g. "gopro-hero11black-linear-5312x2988@29970-no-eis"); may be ""
//	calib_dimension - object {"w":int,"h":int}, the calibration resolution
//	                  (real samples: {5312,2988}, {3840,2160}, {1920,1080})
//	fps             - float calibration frame rate (real samples: 29.97, 29.97003, 23.976025)
//
// calib_dimension + fps pinned by inspecting the committed real fixtures
// testdata/lens/sample_profiles/*.json on 2026-05-17.
type profileJSON struct {
	Name        string `json:"name"`
	CameraBrand string `json:"camera_brand"`
	CameraModel string `json:"camera_model"`
	LensModel   string `json:"lens_model"`
	Identifier  string `json:"identifier"`
	CalibDim    struct {
		W int `json:"w"`
		H int `json:"h"`
	} `json:"calib_dimension"`
	FPS float64 `json:"fps"`
}

// Hit is a single search result. The camera/resolution/fps fields are the
// real lens-profile keys (verified against the committed fixtures) used for
// ranking and surfaced to callers.
type Hit struct {
	// Path is the file-system path to the profile JSON file.
	Path string
	// Display is a human-readable label built from camera_brand, camera_model,
	// and lens_model: "Brand Model (Lens)" or "Brand Model" when lens is empty.
	Display string
	// Identifier is the profile's internal slug (.identifier); may be "".
	Identifier string
	// Brand/Model/Lens are the raw .camera_brand/.camera_model/.lens_model.
	Brand string
	Model string
	Lens  string
	// Width/Height are .calib_dimension.{w,h}; FPS is .fps. Zero when absent.
	Width  int
	Height int
	FPS    float64
}

// Index holds all parsed lens profiles for offline, case-insensitive search.
type Index struct {
	hits       []Hit
	searchable []string // lower-cased concatenation of all searchable fields; parallel to hits
	ids        []string // resource id per hit (identifier field, or filename stem if empty); parallel to hits
	// byID maps a resource id to its index in hits for O(1) ByID lookup.
	// On duplicate ids the FIRST occurrence wins (never overwritten), matching
	// the previous linear-scan first-match behavior.
	byID map[string]int
}

// LoadFromDir walks dir recursively, parses every *.json file as a lens profile,
// and returns an Index. Files that fail JSON parsing are silently skipped so
// that a single malformed file does not abort the entire load.
func LoadFromDir(dir string) (*Index, error) {
	idx := &Index{byID: make(map[string]int)}
	skipped := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}
		h, s, id, parseErr := parseProfile(path)
		if parseErr != nil {
			// Skip malformed/empty file; the profile set is best-effort.
			skipped++
			return nil
		}
		pos := len(idx.hits)
		idx.hits = append(idx.hits, h)
		idx.searchable = append(idx.searchable, s)
		idx.ids = append(idx.ids, id)
		if _, exists := idx.byID[id]; !exists {
			idx.byID[id] = pos // first occurrence wins
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lens: walking %s: %w", dir, err)
	}
	if skipped > 0 {
		// Surface degraded loads (a partial index out of ~12k profiles) so
		// the operator can see it rather than silently losing entries.
		log.Printf("lens: loaded %d profiles from %s, skipped %d malformed/empty", idx.Len(), dir, skipped)
	}
	return idx, nil
}

func parseProfile(path string) (h Hit, searchable, id string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Hit{}, "", "", err
	}
	var p profileJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return Hit{}, "", "", fmt.Errorf("parsing %s: %w", path, err)
	}
	// Skip junk: a profile with BOTH empty name and empty identifier (e.g. an
	// empty "{}" object) carries no searchable identity - indexing it would
	// add an all-spaces searchable string that matches blank queries.
	if p.Name == "" && p.Identifier == "" {
		return Hit{}, "", "", fmt.Errorf("profile %s has empty name and identifier", path)
	}
	display := buildDisplay(p)
	// searchable concatenates all human-visible identity fields lower-cased.
	searchable = strings.ToLower(
		p.CameraBrand + " " + p.CameraModel + " " + p.LensModel + " " + p.Name + " " + p.Identifier,
	)
	// id is the profile's identifier field; some profiles (older calibrator
	// versions) have an empty identifier, so fall back to the filename stem.
	id = p.Identifier
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return Hit{
		Path:       path,
		Display:    display,
		Identifier: p.Identifier,
		Brand:      p.CameraBrand,
		Model:      p.CameraModel,
		Lens:       p.LensModel,
		Width:      p.CalibDim.W,
		Height:     p.CalibDim.H,
		FPS:        p.FPS,
	}, searchable, id, nil
}

func buildDisplay(p profileJSON) string {
	switch {
	case p.LensModel != "":
		return p.CameraBrand + " " + p.CameraModel + " (" + p.LensModel + ")"
	case p.CameraModel != "":
		return p.CameraBrand + " " + p.CameraModel
	default:
		return p.Name
	}
}

// Search returns all hits whose search fields contain query as a
// case-insensitive substring.
func (idx *Index) Search(query string) []Hit {
	q := strings.ToLower(query)
	var out []Hit
	for i, s := range idx.searchable {
		if strings.Contains(s, q) {
			out = append(out, idx.hits[i])
		}
	}
	return out
}

// RankedSearch substring-filters by query (same as Search), then orders the
// matches best-first using the verified real profile keys:
//
//  1. nearest calibration resolution - |w-Width| + |h-Height| ascending
//     (only when width AND height are provided; <=0 means "no preference")
//  2. tie-break: nearest fps - |fps-FPS| ascending (only when fps > 0)
//  3. stable: original index order otherwise
//
// The query already constrains the camera (brand/model/lens/identifier are all
// in the searchable string), so exact-camera candidates are the input set;
// resolution/fps then pick the right shooting-mode variant among multiple
// calibration modes for the same camera.
//
// best is the single top match (nil when there are no matches);
// alternatives is the ranked remainder capped at maxAlternatives.
func (idx *Index) RankedSearch(query string, width, height int, fps float64) (best *Hit, alternatives []Hit) {
	cands := idx.Search(query)
	if len(cands) == 0 {
		return nil, nil
	}
	resDist := func(h Hit) int {
		if width <= 0 || height <= 0 {
			return 0 // no resolution preference → neutral
		}
		return abs(h.Width-width) + abs(h.Height-height)
	}
	fpsDist := func(h Hit) float64 {
		if fps <= 0 {
			return 0 // no fps preference → neutral
		}
		return math.Abs(h.FPS - fps)
	}
	sort.SliceStable(cands, func(i, j int) bool {
		di, dj := resDist(cands[i]), resDist(cands[j])
		if di != dj {
			return di < dj
		}
		return fpsDist(cands[i]) < fpsDist(cands[j])
	})
	best = &cands[0]
	alternatives = cands[1:]
	if len(alternatives) > maxAlternatives {
		alternatives = alternatives[:maxAlternatives]
	}
	return best, alternatives
}

// maxAlternatives caps the ranked remainder so the MCP response stays compact.
const maxAlternatives = 20

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// embeddedProfileBrands is a hand-picked subset of camera brands Gyroflow
// auto-matches to a lens profile from the file's embedded camera identifier -
// NOT the full telemetry-parser supported-formats list; absence here does not
// mean a camera lacks an embedded identifier. For these brands NO explicit
// preset is needed; Gyroflow loads the profile automatically.
//
// Citation (fetched 2026-05-17):
//   - telemetry-parser "Supported formats" (the parser Gyroflow uses to read
//     embedded camera metadata incl. the camera identifier; "xtra" is cited
//     there as "XTRA (Edge, Edge Pro)"):
//     https://raw.githubusercontent.com/AdrianEddy/telemetry-parser/master/README.md
//   - Gyroflow docs, Supported Cameras:
//     https://docs.gyroflow.xyz/app/getting-started/supported-cameras
//
// Empirically confirmed (2026-05-17): a GoPro HERO10 clip rendered correctly
// with NO preset because Gyroflow auto-loaded the lens profile from the
// embedded metadata (verified 2026-05-17 against the committed fixtures:
// .camera_identifier.identifier present).
var embeddedProfileBrands = []string{
	"gopro", "sony", "insta360", "dji", "blackmagic", "red", "canon", "xtra",
}

// cameraBrandAutoLoads reports whether query names a camera brand for which
// Gyroflow auto-loads the lens profile from embedded video metadata.
//
// Matching rule (lowercase, tokenized on non-alphanumeric runes):
//   - a brand matches if ANY token equals the brand string, EXCEPT
//   - the ambiguous English-word brand "red", which matches ONLY when it is
//     the FIRST token.
//
// Rationale - safety asymmetry: a false-positive (reporting "no preset
// needed" when one WAS needed) can silently degrade a render and is unsafe;
// a false-negative (suggesting a preset when auto-load would have worked) is
// safe. "red" occurs incidentally in real camera/lens queries ("Tokina red
// ring lens"), so it is constrained to the first token. The other brands
// (gopro/dji/sony/insta360/blackmagic/canon/xtra) do not occur incidentally,
// so token-anywhere is safe and avoids pure-first-token brittleness
// (e.g. "HERO11 GoPro", "stabilize my GoPro footage" still match).
func cameraBrandAutoLoads(query string) bool {
	tokens := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	for i, tok := range tokens {
		if tok == "red" {
			if i == 0 {
				return true
			}
			continue
		}
		if slices.Contains(embeddedProfileBrands, tok) {
			return true
		}
	}
	return false
}

// GyroflowAutoLoads reports whether query names a camera brand for which
// Gyroflow auto-loads the lens profile from embedded video metadata.
func GyroflowAutoLoads(query string) bool { return cameraBrandAutoLoads(query) }

// AutoLoadAdvice is the guidance surfaced when GyroflowAutoLoads is true.
const AutoLoadAdvice = "gyroflow loads this camera's lens profile automatically " +
	"from the video's embedded metadata; an explicit preset is only needed to override it."

// All returns every hit in the index (the order LoadFromDir walked them).
func (idx *Index) All() []Hit {
	out := make([]Hit, len(idx.hits))
	copy(out, idx.hits)
	return out
}

// Profile is a summary entry pairing a hit with its resource id.
type Profile struct {
	ID      string // resource id (identifier field, or filename stem if empty)
	Path    string // file-system path to the profile JSON
	Display string // human-readable label
}

// Profiles returns a summary of every indexed profile (id, path, display),
// in the order LoadFromDir walked them.
func (idx *Index) Profiles() []Profile {
	out := make([]Profile, 0, len(idx.hits))
	for i, h := range idx.hits {
		out = append(out, Profile{ID: idx.ids[i], Path: h.Path, Display: h.Display})
	}
	return out
}

// Len reports how many profiles are indexed.
func (idx *Index) Len() int { return len(idx.hits) }

// ByID returns the hit whose resource id (the profile identifier field, or the
// filename stem when the identifier is empty) equals id. The bool is false when
// no profile matches.
func (idx *Index) ByID(id string) (Hit, bool) {
	if i, ok := idx.byID[id]; ok {
		return idx.hits[i], true
	}
	return Hit{}, false
}

// lensProfilesURL is the GitHub archive tarball for the gyroflow/lens_profiles
// repository. Default branch verified 2026-05-17 via GitHub API
// (gh api repos/gyroflow/lens_profiles --jq '.default_branch' → "main");
// the refs/heads/master URL also redirects to main.
const lensProfilesURL = "https://github.com/gyroflow/lens_profiles/archive/refs/heads/main.tar.gz"

// Sync downloads the gyroflow/lens_profiles repository as a gzip tarball,
// extracts all *.json files into cacheDir (preserving subdirectory structure),
// and returns a loaded Index. Network or extraction failures are returned as
// wrapped errors; Sync never panics.
func Sync(ctx context.Context, cacheDir string) (*Index, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("lens: creating cache dir: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lensProfilesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("lens: building request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lens: downloading profiles: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lens: unexpected HTTP %d from %s", resp.StatusCode, lensProfilesURL)
	}
	if err := extractTarGz(resp.Body, cacheDir); err != nil {
		return nil, fmt.Errorf("lens: extracting archive: %w", err)
	}
	return LoadFromDir(cacheDir)
}

// extractTarGz unpacks a gzip-compressed tar archive into destDir, stripping
// the top-level directory component and keeping only *.json files.
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip the top-level prefix emitted by GitHub (e.g. "lens_profiles-main/").
		name := hdr.Name
		if idx := strings.Index(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name == "" {
			continue
		}

		dst := filepath.Join(destDir, filepath.FromSlash(name))

		// Zip-slip / tar path-traversal containment: filepath.Join cleans ".."
		// segments, so a crafted "lens_profiles-main/../../../tmp/evil.json"
		// would otherwise escape destDir. Skip (don't write/mkdir) any entry
		// that does not stay within destDir. A single bad entry must not abort
		// a legitimate archive.
		cleanDest := filepath.Clean(destDir)
		cleaned := filepath.Clean(dst)
		if cleaned != cleanDest && !strings.HasPrefix(cleaned, cleanDest+string(os.PathSeparator)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if !strings.HasSuffix(strings.ToLower(dst), ".json") {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			f, err := os.Create(dst)
			if err != nil {
				return err
			}
			// Bound the copy as defense-in-depth against a huge malicious entry.
			_, copyErr := io.Copy(f, io.LimitReader(tr, maxLensFileBytes))
			f.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	return nil
}
