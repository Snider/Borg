// Package smsg - Adaptive Bitrate Streaming (ABR) support
//
// ABR enables multi-bitrate streaming with automatic quality switching based on
// network conditions. Similar to HLS/DASH but with ChaCha20-Poly1305 encryption.
//
// Architecture:
//   - Master manifest (.json) lists available quality variants
//   - Each variant is a standard v3 chunked .smsg file
//   - Same password decrypts all variants (CEK unwrapped once)
//   - Player switches variants at chunk boundaries based on bandwidth
package smsg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const ABRVersion = "abr-v1"

// ABRSafetyFactor is the bandwidth multiplier for variant selection.
// Using 80% of available bandwidth prevents buffering on fluctuating networks.
const ABRSafetyFactor = 0.8

// NewABRManifest creates a new ABR manifest with the given title.
func NewABRManifest(title string) *ABRManifest {
	return &ABRManifest{
		Version:    ABRVersion,
		Title:      title,
		Variants:   make([]Variant, 0),
		DefaultIdx: 0,
	}
}

// AddVariant adds a quality variant to the manifest.
// Variants are automatically sorted by bandwidth (ascending) after adding.
func (m *ABRManifest) AddVariant(v Variant) {
	m.Variants = append(m.Variants, v)
	// Sort by bandwidth ascending (lowest quality first)
	sort.Slice(m.Variants, func(i, j int) bool {
		return m.Variants[i].Bandwidth < m.Variants[j].Bandwidth
	})
	// Update default to 720p if available, otherwise middle variant
	m.DefaultIdx = m.findDefaultVariant()
}

// findDefaultVariant finds the best default variant (prefers 720p).
func (m *ABRManifest) findDefaultVariant() int {
	// Prefer 720p as default
	for i, v := range m.Variants {
		if v.Name == "720p" || v.Height == 720 {
			return i
		}
	}
	// Otherwise use middle variant
	if len(m.Variants) > 0 {
		return len(m.Variants) / 2
	}
	return 0
}

// SelectVariant selects the best variant for the given bandwidth (bits per second).
// Returns the index of the highest quality variant that fits within the bandwidth.
func (m *ABRManifest) SelectVariant(bandwidthBPS int) int {
	safeBandwidth := float64(bandwidthBPS) * ABRSafetyFactor

	// Find highest quality that fits
	selected := 0
	for i, v := range m.Variants {
		if float64(v.Bandwidth) <= safeBandwidth {
			selected = i
		}
	}
	return selected
}

// GetVariant returns the variant at the given index, or nil if out of range.
func (m *ABRManifest) GetVariant(idx int) *Variant {
	if idx < 0 || idx >= len(m.Variants) {
		return nil
	}
	return &m.Variants[idx]
}

// WriteABRManifest writes the ABR manifest to a JSON file.
func WriteABRManifest(manifest *ABRManifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ABR manifest: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write ABR manifest: %w", err)
	}

	return nil
}

// ReadABRManifest reads an ABR manifest from a JSON file.
func ReadABRManifest(path string) (*ABRManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ABR manifest: %w", err)
	}

	return ParseABRManifest(data)
}

// ParseABRManifest parses an ABR manifest from JSON bytes.
func ParseABRManifest(data []byte) (*ABRManifest, error) {
	var manifest ABRManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse ABR manifest: %w", err)
	}

	// Validate version
	if manifest.Version != ABRVersion {
		return nil, fmt.Errorf("unsupported ABR version: %s (expected %s)", manifest.Version, ABRVersion)
	}

	return &manifest, nil
}

// VariantFromSMSG creates a Variant from an existing .smsg file.
// It reads the header to extract chunk count and file size.
func VariantFromSMSG(name string, bandwidth, width, height int, smsgPath string) (*Variant, error) {
	// Read file to get size and chunk info
	data, err := os.ReadFile(smsgPath)
	if err != nil {
		return nil, fmt.Errorf("read smsg file: %w", err)
	}

	// Get header to extract chunk count
	header, err := GetV3Header(data)
	if err != nil {
		return nil, fmt.Errorf("parse smsg header: %w", err)
	}

	chunkCount := 0
	if header.Chunked != nil {
		chunkCount = header.Chunked.TotalChunks
	}

	return &Variant{
		Name:       name,
		Bandwidth:  bandwidth,
		Width:      width,
		Height:     height,
		Codecs:     "avc1.640028,mp4a.40.2", // Default H.264 + AAC
		URL:        filepath.Base(smsgPath),
		ChunkCount: chunkCount,
		FileSize:   int64(len(data)),
	}, nil
}

// ABRBandwidthEstimator tracks download speeds for adaptive quality selection.
type ABRBandwidthEstimator struct {
	samples    []int // bandwidth samples in bps
	maxSamples int
}

// NewABRBandwidthEstimator creates a new bandwidth estimator.
func NewABRBandwidthEstimator(maxSamples int) *ABRBandwidthEstimator {
	if maxSamples <= 0 {
		maxSamples = 10
	}
	return &ABRBandwidthEstimator{
		samples:    make([]int, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// RecordSample records a bandwidth sample from a download.
// bytes is the number of bytes downloaded, durationMs is the time in milliseconds.
func (e *ABRBandwidthEstimator) RecordSample(bytes int, durationMs int) {
	if durationMs <= 0 {
		return
	}
	// Calculate bits per second: (bytes * 8 * 1000) / durationMs
	bps := (bytes * 8 * 1000) / durationMs
	e.samples = append(e.samples, bps)
	if len(e.samples) > e.maxSamples {
		e.samples = e.samples[1:]
	}
}

// Estimate returns the estimated bandwidth in bits per second.
// Uses average of recent samples, or 1 Mbps default if no samples.
func (e *ABRBandwidthEstimator) Estimate() int {
	if len(e.samples) == 0 {
		return 1000000 // 1 Mbps default
	}

	// Use average of last 3 samples (or all if fewer)
	count := 3
	if len(e.samples) < count {
		count = len(e.samples)
	}
	recent := e.samples[len(e.samples)-count:]

	sum := 0
	for _, s := range recent {
		sum += s
	}
	return sum / count
}
