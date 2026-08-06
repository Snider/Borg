// +build !dappfm

package player

import (
	"embed"
	"io/fs"
)

// Assets embeds all frontend files for the media player (except demo track)
// To build with full assets including demo track, use: go build -tags dappfm
//
//go:embed frontend/index.html
//go:embed frontend/wasm_exec.js
//go:embed frontend/stmf.wasm
var assets embed.FS

// Assets returns the embedded filesystem with frontend/ prefix stripped
var Assets fs.FS

func init() {
	var err error
	Assets, err = fs.Sub(assets, "frontend")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}
}

// GetDemoTrack returns an error since demo track is not available in stub build
func GetDemoTrack() ([]byte, error) {
	return nil, fs.ErrNotExist
}

// GetIndex returns the main HTML page
func GetIndex() ([]byte, error) {
	return fs.ReadFile(Assets, "index.html")
}
