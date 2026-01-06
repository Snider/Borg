package player

import (
	"embed"
	"io/fs"
)

// Assets embeds all frontend files for the media player
// These are served both by Wails (memory) and HTTP (fallback)
//
//go:embed frontend/index.html
//go:embed frontend/wasm_exec.js
//go:embed frontend/stmf.wasm
//go:embed frontend/demo-track.smsg
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

// GetDemoTrack returns the embedded demo track content
func GetDemoTrack() ([]byte, error) {
	return fs.ReadFile(Assets, "demo-track.smsg")
}

// GetIndex returns the main HTML page
func GetIndex() ([]byte, error) {
	return fs.ReadFile(Assets, "index.html")
}
