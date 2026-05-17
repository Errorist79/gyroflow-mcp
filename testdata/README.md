# testdata

## Committed fixtures (small, tracked)

| File | Description |
|---|---|
| `help_output.txt` | Real `gyroflow --help` output (v1.6.3) - used to verify CLI flag names and -p/-s JSON key schema |
| `version_output.txt` | Real `gyroflow --version` output (v1.6.3) - used to verify version-parsing regex |
| `stdout_progress_sample.txt` | Real `gyroflow --stdout-progress` output captured from `sample_gyro.mp4` - pinned regex source |

## Git-excluded media (`testdata/vids/`, NOT committed)

All test/sample/render media lives under `testdata/vids/` (the whole directory
is git-ignored - multi-MB, never committed). Integration tests check for the
input fixture's presence and call `t.Skip` when absent, so unit CI stays fast
and dependency-free.

### `testdata/vids/sample_gyro.mp4`

**Source:** Gyroflow official "Test Files"
- Docs page: https://docs.gyroflow.xyz/app/readme/test-files
- Official Google Drive folder: https://drive.google.com/drive/folders/1sbZiLN5-sv_sGul1E_DUOluB5OMHfySh
- File: "GoPro Hero 10.MP4" - save locally as `testdata/vids/sample_gyro.mp4` (~124 MB)

**Camera:** GoPro Hero 10 Black
**Gyro data:** Embedded GPMF gyroscope (no external gyro file needed)
**Purpose:** Integration tests that run the real `gyroflow` binary

### To obtain

Download "GoPro Hero 10.MP4" from the official Google Drive folder above and
save it as `testdata/vids/sample_gyro.mp4`. Render outputs and any ad-hoc
comparison clips also belong under `testdata/vids/` and are ignored there.
