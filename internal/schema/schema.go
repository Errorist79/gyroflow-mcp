// Package schema holds gyroflow JSON parameter schemas verified against
// the real `gyroflow --help` output (see testdata/help_output.txt).
//
// -p example from --help: "{ 'codec': 'H.265/HEVC', 'bitrate': 150, 'use_gpu': true, 'audio': true }"
// -s example from --help: "{ 'search_size': 3, 'processing_resolution': 720 }"
package schema

// OutParams represents the gyroflow -p / --out-params JSON object.
// Field names verified verbatim against gyroflow 1.6.3 --help.
type OutParams struct {
	Codec   string `json:"codec,omitempty"`
	Bitrate int    `json:"bitrate,omitempty"`
	UseGPU  bool   `json:"use_gpu,omitempty"`
	Audio   bool   `json:"audio,omitempty"`
}

// SyncParams represents the gyroflow -s / --sync-params JSON object.
// Field names verified verbatim against gyroflow 1.6.3 --help.
type SyncParams struct {
	SearchSize           int `json:"search_size,omitempty"`
	ProcessingResolution int `json:"processing_resolution,omitempty"`
}
